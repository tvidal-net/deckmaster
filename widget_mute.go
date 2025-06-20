package main

import (
	"github.com/godbus/dbus/v5"
	"image"
)

const (
	StreamConfig    = "stream"
	DisabledConfig  = "disabled"
	MicStreamConfig = "mic"
)

var (
	cnn = NewKMixerConnection()
)

type MuteWidget struct {
	*ButtonWidget

	disabled image.Image
	playback bool
	enabled  bool
	state    bool
}

func enabled(playback bool) (bool, error) {
	if playback {
		return cnn.isPlaybackMuted()
	} else {
		return cnn.isMicMuted()
	}
}

func NewMuteWidget(bw *BaseWidget, opts WidgetConfig) (*MuteWidget, error) {
	var disabled, stream string
	_ = ConfigValue(opts.Config[DisabledConfig], &disabled)
	_ = ConfigValue(opts.Config[StreamConfig], &stream)
	button, err := NewButtonWidget(bw, opts)
	if err != nil {
		return nil, err
	}
	isPlaybackMute := stream != MicStreamConfig
	initialState, _ := enabled(isPlaybackMute)
	w := &MuteWidget{
		ButtonWidget: button,
		playback:     isPlaybackMute,
		enabled:      initialState,
		state:        !initialState,
	}
	if err := w.LoadImage(&w.disabled, disabled); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *MuteWidget) Signal(signal *dbus.Signal) {
	if isControlChanged(signal) && w.playback == isPlayback(signal) {
		enabled, err := enabled(w.playback)
		if err != nil {
			errorf(err)
		}
		w.enabled = enabled
	}
}

func (w *MuteWidget) RequiresUpdate() bool {
	return w.enabled != w.state
}

func (w *MuteWidget) Update() error {
	if w.enabled != w.state {
		var icon image.Image
		if w.enabled {
			icon = w.icon
		} else {
			icon = w.disabled
		}
		w.state = w.enabled
		return w.RenderButton(icon)
	}
	return nil
}

func (w *MuteWidget) TriggerAction(hold bool) {
	var err error
	if !hold {
		if w.playback {
			err = cnn.TogglePlaybackMute()
		} else {
			err = cnn.ToggleMicMute()
		}
	}
	if err != nil {
		errorf(err)
	}
}
