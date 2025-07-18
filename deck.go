package main

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/muesli/streamdeck"
)

var (
	SPACES = regexp.MustCompile(`\s+`)
	PATH   = strings.Split(os.Getenv("PATH"), ":")
)

// Deck is a set of widgets.
type Deck struct {
	File       string
	Background image.Image
	Widgets    map[uint8]Widget
}

func expandExecutable(exe string) string {
	for _, base := range PATH {
		cmd := filepath.Join(base, exe)
		s, e := os.Stat(cmd)
		if e != nil || s.IsDir() {
			continue
		}
		fileMode := s.Mode()
		if fileMode&0111 != 0 {
			return cmd
		}
	}
	return exe
}

// LoadDeck loads a deck configuration.
func LoadDeck(dev *streamdeck.Device, base string, deck string) (*Deck, error) {
	path, err := expandPath(base, deck)
	if err != nil {
		return nil, err
	}
	verboseLog("Loading deck: %s", path)

	dc, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}

	d := Deck{
		Widgets: make(map[uint8]Widget),
		File:    path,
	}
	if dc.Background != "" {
		bgpath, err := expandPath(filepath.Dir(path), dc.Background)
		if err != nil {
			return nil, err
		}
		if err := d.loadBackground(dev, bgpath); err != nil {
			return nil, err
		}
	}

	keyMap := map[uint8]KeyConfig{}
	for _, k := range dc.Keys {
		keyMap[k.Index] = k
	}

	for i := uint8(0); i < dev.Keys; i++ {
		bg := d.backgroundForKey(dev, i)

		var w Widget
		if k, found := keyMap[i]; found {
			w, err = NewWidget(dev, filepath.Dir(path), k, bg)
			if err != nil {
				return nil, err
			}
		} else {
			w = NewBaseWidget(dev, filepath.Dir(path), i, nil, nil, bg)
		}

		d.Widgets[i] = w
	}

	return &d, nil
}

// loads a background image.
func (d *Deck) loadBackground(dev *streamdeck.Device, bg string) error {
	f, err := os.Open(bg)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	background, _, err := image.Decode(f)
	if err != nil {
		return err
	}

	rows := int(dev.Rows)
	cols := int(dev.Columns)
	padding := int(dev.Padding)
	pixels := int(dev.Pixels)

	width := cols*pixels + (cols-1)*padding
	height := rows*pixels + (rows-1)*padding
	if background.Bounds().Dx() != width ||
		background.Bounds().Dy() != height {
		return fmt.Errorf("supplied background image has wrong dimensions, expected %dx%d pixels", width, height)
	}

	d.Background = background
	return nil
}

// returns the background image for an individual key.
func (d *Deck) backgroundForKey(dev *streamdeck.Device, key uint8) image.Image {
	padding := int(dev.Padding)
	pixels := int(dev.Pixels)
	bg := image.NewRGBA(image.Rect(0, 0, pixels, pixels))

	if d.Background != nil {
		startx := int(key%dev.Columns) * (pixels + padding)
		starty := int(key/dev.Columns) * (pixels + padding)
		draw.Draw(bg, bg.Bounds(), d.Background, image.Point{startx, starty}, draw.Src)
	}

	return bg
}

// handles keypress with delay.
func emulateKeyPressWithDelay(keys string) {
	kd := strings.Split(keys, "+")
	emulateKeyPress(kd[0])
	if len(kd) == 1 {
		return
	}

	// optional delay
	if delay, err := strconv.Atoi(strings.TrimSpace(kd[1])); err == nil {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
}

// emulates a range of key presses.
func emulateKeyPresses(keys string) {
	for _, kp := range strings.Split(keys, "/") {
		emulateKeyPressWithDelay(kp)
	}
}

// emulates a (multi-)key press.
func emulateKeyPress(keys string) {
	if keyboard == nil {
		errorLogF("Keyboard emulation is disabled!")
		return
	}

	kk := strings.Split(keys, "-")
	for i, k := range kk {
		k = formatKeycodes(strings.TrimSpace(k))
		kc, e := strconv.Atoi(k)
		if e != nil {
			errorLog(e, "%s is not a valid keycode", k)
		}

		if i+1 < len(kk) {
			_ = keyboard.KeyDown(kc)
			defer keyboard.KeyUp(kc) //nolint:errcheck
		} else {
			_ = keyboard.KeyPress(kc)
		}
	}
}

// emulates a clipboard paste.
func emulateClipboard(text string) {
	errorLog(clipboard.WriteAll(text), "failed to paste from the Clipboard")

	// paste the string
	emulateKeyPress("29-47") // ctrl-v
}

// executes a dbus method.
func executeDBusMethod(config *DBusConfig) {
	if e := CallDBus(config.Object, config.Path, config.Method, config.Value); e != nil {
		errorLog(e, "DBus call failed %+v", config)
	}
}

// executes a command.
func executeCommand(cmd string) error {
	args := SPACES.Split(cmd, -1)
	exe := expandExecutable(args[0])

	c := exec.Command(exe, args[1:]...) //nolint:gosec
	if *verboseConfig {
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
	}

	if e := c.Start(); e != nil {
		errorLogF("failed to execute '%s %s'", exe, args[1:])
		return e
	}
	return c.Wait()
}

// triggerAction triggers an action.
func (d *Deck) triggerAction(dev *streamdeck.Device, index uint8, hold bool) {
	var w = d.Widgets[index]
	w.TriggerAction(hold)

	var a *ActionConfig
	if hold {
		a = w.ActionHold()
	} else {
		a = w.Action()
	}

	if a == nil {
		return
	}
	if a.Deck != "" {
		newDeck, err := LoadDeck(dev, filepath.Dir(d.File), a.Deck)
		if err != nil {
			errorLog(err, "Failed to load deck %s", a.Deck)
			return
		}
		if err := dev.Clear(); err != nil {
			fatal(err)
			return
		}

		deck = newDeck
		deck.updateWidgets()
	}
	if a.Keycode != "" {
		emulateKeyPresses(a.Keycode)
	}
	if a.Paste != "" {
		emulateClipboard(a.Paste)
	}
	if a.DBus.Method != "" {
		executeDBusMethod(&a.DBus)
	}
	if a.Exec != "" {
		go func(a *ActionConfig) {
			errorLog(executeCommand(a.Exec), "failed to execute command")
		}(a)
	}
	if a.Device != "" {
		switch {
		case a.Device == "sleep":
			if err := dev.Sleep(); err != nil {
				fatal(err)
			}

		case strings.HasPrefix(a.Device, "brightness"):
			d.adjustBrightness(dev, strings.TrimPrefix(a.Device, "brightness"))

		default:
			errorLogF("Unrecognized special action: %s", a.Device)
		}
	}
}

// updateWidgets updates/repaints all the widgets.
func (d *Deck) updateWidgets() {
	for _, w := range d.Widgets {
		if !w.RequiresUpdate() {
			continue
		}

		// fmt.Println("Repaint", w.Key())
		if err := w.Update(); err != nil {
			fatal(err)
		}
	}
}

// adjustBrightness adjusts the brightness.
func (d *Deck) adjustBrightness(dev *streamdeck.Device, value string) {
	if len(value) == 0 {
		errorLogF("no brightness value specified")
		return
	}

	v := int64(math.MinInt64)
	if len(value) > 1 {
		nv, err := strconv.ParseInt(value[1:], 10, 64)
		if err == nil {
			v = nv
		}
	}

	switch value[0] {
	case '=': // brightness=[n]:
	case '-': // brightness-[n]:
		if v == math.MinInt64 {
			v = 10
		}
		v = int64(*brightnessConfig) - v
	case '+': // brightness+[n]:
		if v == math.MinInt64 {
			v = 10
		}
		v = int64(*brightnessConfig) + v
	default:
		v = math.MinInt64
	}

	if v == math.MinInt64 {
		errorLogF("could not grok the brightness from value '%s'", value)
		return
	}

	if v < 1 {
		v = 1
	} else if v > 100 {
		v = 100
	}
	if err := dev.SetBrightness(uint8(v)); err != nil {
		fatal(err)
	}

	*brightnessConfig = uint(v)
}
