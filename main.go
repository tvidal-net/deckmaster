package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/bendahl/uinput"
	"github.com/godbus/dbus/v5"
	"github.com/mitchellh/go-homedir"
	"github.com/muesli/streamdeck"
)

var (
	// Version contains the application version number. It's set via ldflags
	// when building.
	Version = ""

	// CommitSHA contains the SHA of the commit that this application was built
	// against. It's set via ldflags when building.
	CommitSHA = ""

	deck *Deck

	dbusConn *dbus.Conn
	keyboard uinput.Keyboard
	shutdown = make(chan error)

	pa            *PulseAudio
	xorg          *Xorg
	recentWindows []Window

	deckFileConfig   = flag.String("deck", "main.deck", "path to deck config file")
	deviceConfig     = flag.String("device", "", "which device to use (serial number)")
	brightnessConfig = flag.Uint("brightness", 80, "brightness in percent")
	sleepConfig      = flag.String("sleep", "", "sleep timeout")
	verboseConfig    = flag.Bool("verbose", false, "verbose output")
	versionConfig    = flag.Bool("version", false, "display version")
)

const (
	fadeDuration      = 250 * time.Millisecond
	longPressDuration = 350 * time.Millisecond
)

func errorLog(e error) {
	if e != nil {
		errorLogF("ERROR: %s\n", e.Error())
	}
}

func errorLogF(format string, args ...any) {
	errorLog(fmt.Errorf(format, args...))
}

func fatal(e error) {
	go func() { shutdown <- e }()
}

func verboseLog(format string, a ...interface{}) {
	if *verboseConfig {
		fmt.Printf(format+"\n", a...)
	}
}

func expandPath(base, path string) (string, error) {
	var err error
	path, err = homedir.Expand(path)
	if err != nil {
		return "", err
	}
	if base == "" {
		return path, nil
	}

	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}

	return filepath.Abs(path)
}

func eventLoop(dev *streamdeck.Device, tch chan interface{}) error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)

	var keyStates sync.Map
	keyTimestamps := make(map[uint8]time.Time)

	var audioWidgets []AudioChangedMonitor
	var muteWidgets []MuteChangedMonitor
	for _, widget := range deck.Widgets {
		muteWidget, success := widget.(MuteChangedMonitor)
		if success {
			muteWidgets = append(muteWidgets, muteWidget)
		}

		audioWidget, success := widget.(AudioChangedMonitor)
		if success {
			audioWidgets = append(audioWidgets, audioWidget)
		}
	}
	go pa.Start()

	kch, err := dev.ReadKeys()
	if err != nil {
		return err
	}
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			deck.updateWidgets()

		case k, ok := <-kch:
			if !ok {
				if err = dev.Open(); err != nil {
					return err
				}
				continue
			}

			var state bool
			if ks, ok := keyStates.Load(k.Index); ok {
				state = ks.(bool)
			}
			keyStates.Store(k.Index, k.Pressed)

			if state && !k.Pressed {
				// key was released
				if time.Since(keyTimestamps[k.Index]) < longPressDuration {
					verboseLog("Triggering short action for key %d", k.Index)
					deck.triggerAction(dev, k.Index, false)
				}
			}
			if !state && k.Pressed {
				// key was pressed
				go func() {
					// launch timer to observe KeyState
					time.Sleep(longPressDuration)

					if state, ok := keyStates.Load(k.Index); ok && state.(bool) {
						// key still pressed
						verboseLog("Triggering long action for key %d", k.Index)
						deck.triggerAction(dev, k.Index, true)
					}
				}()
			}
			keyTimestamps[k.Index] = time.Now()

		case changeType := <-pa.Updates():
			switch changeType {
			case SinkMuteChanged, SourceMuteChanged:
				playback := changeType == SinkMuteChanged
				for _, w := range muteWidgets {
					w.MuteChanged(playback)
				}

			case SinkChanged, SourceChanged:
				for _, w := range audioWidgets {
					w.AudioStreamChanged(changeType)
				}
			}

		case e := <-tch:
			switch event := e.(type) {
			case WindowClosedEvent:
				handleWindowClosed(event)

			case ActiveWindowChangedEvent:
				handleActiveWindowChanged(dev, event)
			}

		case err := <-shutdown:
			return err

		case <-hup:
			verboseLog("Received SIGHUP, reloading configuration...")

			nd, err := LoadDeck(dev, ".", deck.File)
			if err != nil {
				verboseLog("The new configuration is not valid, keeping the current one.")
				errorLogF("Configuration Error: %s", err.Error())
				continue
			}

			deck = nd
			deck.updateWidgets()

		case <-sigs:
			fmt.Println("Shutting down...")
			return nil
		}
	}
}

func closeDevice(dev *streamdeck.Device) {
	if err := dev.Reset(); err != nil {
		errorLogF("Unable to reset Stream Deck")
	}
	if err := dev.Clear(); err != nil {
		errorLogF("Unable to clear the Stream Deck")
	}
	if err := dev.Sleep(); err != nil {
		errorLogF("Unable to sleep the Stream Deck")
	}
	if err := dev.Close(); err != nil {
		errorLogF("Unable to close Stream Deck")
	}
}

func initDevice() (*streamdeck.Device, error) {
	d, err := streamdeck.Devices()
	if err != nil {
		return nil, err
	}
	if len(d) == 0 {
		return nil, fmt.Errorf("no Stream Deck devices found")
	}

	dev := d[0]
	if len(*deviceConfig) > 0 {
		found := false
		for _, v := range d {
			if v.Serial == *deviceConfig {
				dev = v
				found = true
				break
			}
		}
		if !found {
			errorLogF("Can't find device. Available devices:")
			for _, v := range d {
				errorLogF("Serial %s (%d buttons)", v.Serial, dev.Keys)
			}
			os.Exit(1)
		}
	}

	if err := dev.Open(); err != nil {
		return nil, err
	}
	ver, err := dev.FirmwareVersion()
	if err != nil {
		return &dev, err
	}
	verboseLog("Found device with serial %s (%d buttons, firmware %s)",
		dev.Serial, dev.Keys, ver)

	if err := dev.Reset(); err != nil {
		return &dev, err
	}

	if *brightnessConfig > 100 {
		*brightnessConfig = 100
	}
	if err = dev.SetBrightness(uint8(*brightnessConfig)); err != nil {
		return &dev, err
	}

	dev.SetSleepFadeDuration(fadeDuration)
	if len(*sleepConfig) > 0 {
		timeout, err := time.ParseDuration(*sleepConfig)
		if err != nil {
			return &dev, err
		}

		dev.SetSleepTimeout(timeout)
	}

	return &dev, nil
}

func run() error {
	// initialize device
	dev, err := initDevice()
	if dev != nil {
		defer closeDevice(dev)
	}
	if err != nil {
		return fmt.Errorf("Unable to initialize Stream Deck: %s", err)
	}

	// initialize dbus connection
	dbusConn, err = dbus.SessionBus()
	if err != nil {
		return fmt.Errorf("Unable to connect to dbus: %s", err)
	}

	// initialize xorg connection and track window focus
	tch := make(chan interface{})
	xorg, err = Connect(os.Getenv("DISPLAY"))
	if err == nil {
		defer xorg.Close()
		xorg.TrackWindows(tch, time.Second)
	} else {
		errorLogF("Could not connect to X server: %s", err.Error())
		errorLogF("Tracking window manager will be disabled!")
	}

	// initialize virtual keyboard
	keyboard, err = uinput.CreateKeyboard("/dev/uinput", []byte("Deckmaster"))
	if err != nil {
		errorLogF("Could not create virtual input device (/dev/uinput): %s", err.Error())
		errorLogF("Emulating keyboard events will be disabled!")
	} else {
		defer keyboard.Close() //nolint:errcheck
	}

	// initialize PulseAudio
	newPulseAudio, err := NewPulseAudio()
	if err != nil {
		return err
	}
	pa = newPulseAudio
	defer pa.Close()

	// load deck
	deck, err = LoadDeck(dev, ".", *deckFileConfig)
	if err != nil {
		return fmt.Errorf("Can't load deck: %s", err)
	}
	deck.updateWidgets()

	return eventLoop(dev, tch)
}

func main() {
	flag.Parse()

	if *versionConfig {
		if len(CommitSHA) > 7 {
			CommitSHA = CommitSHA[:7]
		}
		if Version == "" {
			Version = "(built from source)"
		}

		fmt.Printf("deckmaster %s", Version)
		if len(CommitSHA) > 0 {
			fmt.Printf(" (%s)", CommitSHA)
		}

		fmt.Println()
		os.Exit(0)
	}

	if err := run(); err != nil {
		errorLog(err)
		os.Exit(1)
	}
}
