package main

import (
	"image"
)

const (
	StreamConfig    = "stream"
	MutedConfig     = "muted"
	MicStreamConfig = "mic"
)

type MuteWidget struct {
	*ButtonWidget

	muted    image.Image
	playback bool
}

type MuteChangedMonitor interface {
	MuteChanged(playback bool)
}

func NewMuteWidget(bw *BaseWidget, opts WidgetConfig) (*MuteWidget, error) {
	var muted, stream string
	_ = ConfigValue(opts.Config[MutedConfig], &muted)
	_ = ConfigValue(opts.Config[StreamConfig], &stream)
	button, err := NewButtonWidget(bw, opts)
	if err != nil {
		return nil, err
	}
	isPlayback := stream != MicStreamConfig
	w := &MuteWidget{
		ButtonWidget: button,
		playback:     isPlayback,
	}
	if err := w.LoadImage(&w.muted, muted); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *MuteWidget) Update() error {
	if pa.Muted(w.playback) {
		return w.RenderButton(w.muted)
	} else {
		return w.RenderButton(w.icon)
	}
}

func (w *MuteWidget) TriggerAction(hold bool) {
	if !hold {
		err := pa.ToggleMute(w.playback)
		if err != nil {
			errorf(err)
		}
	}
}

func (w *MuteWidget) MuteChanged(playback bool) {
	if playback == w.playback {
		err := w.Update()
		if err != nil {
			errorf(err)
		}
	}
}
