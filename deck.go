package main

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/godbus/dbus/v5"
	"github.com/muesli/streamdeck"
)

// Deck is a set of widgets.
type Deck struct {
	File       string
	Background image.Image
	Widgets    []Widget
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
		File: path,
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

		d.Widgets = append(d.Widgets, w)
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
		kc, err := strconv.Atoi(k)
		if err != nil {
			errorLogF("%s is not a valid keycode: %s", k, err.Error())
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
	err := clipboard.WriteAll(text)
	if err != nil {
		errorLogF("Pasting to clipboard failed: %s", err.Error())
	}

	// paste the string
	emulateKeyPress("29-47") // ctrl-v
}

// executes a dbus method.
func executeDBusMethod(object, path, method, args string) {
	call := dbusConn.Object(object, dbus.ObjectPath(path)).Call(method, 0, args)
	if call.Err != nil {
		errorLogF("dbus call failed: %s", call.Err.Error())
	}
}

// executes a command.
func executeCommand(cmd string) error {
	exp, err := expandPath("", cmd)
	if err == nil {
		cmd = exp
	}
	args := strings.Split(cmd, " ")

	c := exec.Command(args[0], args[1:]...) //nolint:gosec
	if *verboseConfig {
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
	}

	if err := c.Start(); err != nil {
		errorLogF("Command failed: %s", err.Error())
		return err
	}
	return c.Wait()
}

// triggerAction triggers an action.
func (d *Deck) triggerAction(dev *streamdeck.Device, index uint8, hold bool) {
	for _, w := range d.Widgets {
		if w.Key() != index {
			continue
		}
		w.TriggerAction(hold)

		var a *ActionConfig
		if hold {
			a = w.ActionHold()
		} else {
			a = w.Action()
		}

		if a == nil {
			continue
		}
		if a.Deck != "" {
			newDeck, err := LoadDeck(dev, filepath.Dir(d.File), a.Deck)
			if err != nil {
				errorLogF("Can't load deck: %s", err.Error())
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
			executeDBusMethod(a.DBus.Object, a.DBus.Path, a.DBus.Method, a.DBus.Value)
		}
		if a.Exec != "" {
			go executeCommand(a.Exec)
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
		errorLogF("No brightness value specified")
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
		errorLogF("Could not grok the brightness from value '%s'", value)
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
