package main

import (
	"image"
	"strings"
)

const (
	AltImageConfig   = "alt"
	MainStreamConfig = "main"
	AltStreamConfig  = "stream"
)

type AudioWidget struct {
	*ButtonWidget

	alt        image.Image
	mainStream []string
	altStream  []string
}

type AudioChangedMonitor interface {
	AudioChanged(changeType ChangeType)
}

func NewAudioWidget(bw *BaseWidget, opts WidgetConfig) (*AudioWidget, error) {
	button, err := NewButtonWidget(bw, opts)
	if err != nil {
		return nil, err
	}

	var mainStreamConfig, altStreamConfig, altImageConfig string
	_ = ConfigValue(opts.Config[MainStreamConfig], &mainStreamConfig)
	_ = ConfigValue(opts.Config[AltStreamConfig], &altStreamConfig)
	w := &AudioWidget{
		ButtonWidget: button,
		mainStream:   strings.Split(mainStreamConfig, ","),
		altStream:    strings.Split(altStreamConfig, ","),
	}
	_ = ConfigValue(opts.Config[AltImageConfig], &altImageConfig)
	if err := w.LoadImage(&w.alt, altImageConfig); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *AudioWidget) MainSourceStream() string {
	if len(w.mainStream) > 0 {
		return w.mainStream[0]
	}
	return ""
}

func (w *AudioWidget) MainSinkStream() string {
	if len(w.mainStream) > 1 {
		return w.mainStream[1]
	}
	return w.MainSourceStream()
}

func (w *AudioWidget) AltSourceStream() string {
	if len(w.altStream) > 0 {
		return w.altStream[0]
	}
	return ""
}

func (w *AudioWidget) AltSinkStream() string {
	if len(w.altStream) > 1 {
		return w.altStream[1]
	}
	return w.AltSourceStream()
}

func (w *AudioWidget) IsMainStreamDefault() bool {
	sinkName := w.MainSinkStream()
	if sinkName == "" {
		return !strings.Contains(pa.CurrentSinkName(), w.AltSinkStream())
	}
	return strings.Contains(pa.CurrentSinkName(), sinkName)
}

func (w *AudioWidget) SetSinkStream(alt bool) {
	if alt {
		errorf(pa.SetSink(w.AltSinkStream()))
	} else {
		errorf(pa.SetSink(w.MainSinkStream()))
	}
}

func (w *AudioWidget) SetSourceStream(alt bool) {
	if alt {
		errorf(pa.SetSource(w.AltSourceStream()))
	} else {
		errorf(pa.SetSource(w.MainSourceStream()))
	}
	errorf(w.Update())
}

func (w *AudioWidget) Update() error {
	if w.IsMainStreamDefault() {
		return w.RenderButton(w.icon)
	} else {
		return w.RenderButton(w.alt)
	}
}

func (w *AudioWidget) TriggerAction(hold bool) {
	if !hold {
		w.SetSinkStream(w.IsMainStreamDefault())
	}
}

func (w *AudioWidget) AudioChanged(changeType ChangeType) {
	if changeType == SinkChanged {
		verbosef("SinkChanged")
		w.SetSourceStream(!w.IsMainStreamDefault())
	} else {
		verbosef("SourceChanged")
		errorf(w.Update())
	}
}
