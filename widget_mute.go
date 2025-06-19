package main

import (
	"github.com/godbus/dbus/v5"
	"image"
)

var (
	cnn = NewKMixerConnection()
)

type MuteWidget struct {
	*ButtonWidget

	disabled image.Image
	playback bool
	state    bool
}

func enabled(playback bool) (bool, error) {
	if playback {
		return cnn.isPlaybackMuted()
	} else {
		return cnn.isMicMuted()
	}
}

func (w *MuteWidget) Signal(signal *dbus.Signal) {
	if isControlChanged(signal) {
		err := w.Update()
		if err != nil {
			errorf(err)
		}
	}
}

func NewMuteWidget(bw *BaseWidget, opts WidgetConfig) (*MuteWidget, error) {
	var disabled, stream string
	_ = ConfigValue(opts.Config["disabled"], &disabled)
	_ = ConfigValue(opts.Config["stream"], &stream)
	button, err := NewButtonWidget(bw, opts)
	if err != nil {
		return nil, err
	}
	isPlaybackMute := stream != "mic"
	initialState, _ := enabled(isPlaybackMute)
	w := &MuteWidget{
		ButtonWidget: button,
		playback:     isPlaybackMute,
		state:        !initialState,
	}
	if err := w.LoadImage(&w.disabled, disabled); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *MuteWidget) Update() error {
	enabled, err := enabled(w.playback)
	if err != nil {
		return err
	}
	if enabled != w.state {
		var icon image.Image
		if enabled {
			icon = w.icon
		} else {
			icon = w.disabled
		}
		w.state = enabled
		return w.RenderButton(icon)
	}
	return nil
}
