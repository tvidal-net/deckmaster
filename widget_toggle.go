package main

import (
	"image"
	"time"
)

type ToggleWidget struct {
	*ButtonWidget

	disabled image.Image
	state    string
	active   bool
}

func NewToggleWidget(bw *BaseWidget, opts WidgetConfig) (*ToggleWidget, error) {
	var disabled, state string
	_ = ConfigValue(opts.Config["disabled"], &disabled)
	_ = ConfigValue(opts.Config["state"], &state)

	button, err := NewButtonWidget(bw, opts)
	if err != nil {
		return nil, err
	}
	w := &ToggleWidget{
		ButtonWidget: button,
		state:        state,
	}
	if err := w.LoadImage(&w.disabled, disabled); err != nil {
		return nil, err
	}

	w.active = w.CheckButtonState()
	return w, nil
}

func (w *ToggleWidget) Icon() image.Image {
	if w.active || w.disabled == nil {
		return w.icon
	} else {
		return w.disabled
	}
}

func (w *ToggleWidget) Update() error {
	return w.RenderButton(w.Icon())
}

// TriggerAction default action is to toggle the button image
func (w *ToggleWidget) TriggerAction(hold bool) {
	if w.state != "" {
		go UpdateButtonState(w)
	}
}

func UpdateButtonState(w *ToggleWidget) {
	if w.interval > 0 {
		time.Sleep(w.interval)
	}
	if w.active != w.CheckButtonState() {
		w.active = !w.active
		w.Update()
	}
}

func (w *ToggleWidget) CheckButtonState() bool {
	if w.state != "" {
		verbosef("checking for state of button %d with '%s'", w.key, w.state)
		return executeCommand(w.state) == nil
	}
	return true
}
