package main

import (
	"image"
	"image/color"
	"time"
)

// ButtonWidget is a simple widget displaying an icon and/or label.
type ButtonWidget struct {
	*BaseWidget

	icon     image.Image
	disabled image.Image
	label    string
	fontsize float64
	color    color.Color
	flatten  bool
}

// NewButtonWidget returns a new ButtonWidget.
func NewButtonWidget(bw *BaseWidget, opts WidgetConfig) (*ButtonWidget, error) {
	bw.setInterval(time.Duration(opts.Interval)*time.Millisecond, 0)

	var icon, disabled, label string
	_ = ConfigValue(opts.Config["icon"], &icon)
	_ = ConfigValue(opts.Config["disabled"], &disabled)
	_ = ConfigValue(opts.Config["label"], &label)
	var fontsize float64
	_ = ConfigValue(opts.Config["fontsize"], &fontsize)
	var color color.Color
	_ = ConfigValue(opts.Config["color"], &color)
	var flatten bool
	_ = ConfigValue(opts.Config["flatten"], &flatten)

	if color == nil {
		color = DefaultColor
	}

	w := &ButtonWidget{
		BaseWidget: bw,
		label:      label,
		fontsize:   fontsize,
		color:      color,
		flatten:    flatten,
	}
	if err := w.LoadImage(&w.icon, icon); err != nil {
		return nil, err
	}
	if err := w.LoadImage(&w.disabled, disabled); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *ButtonWidget) LoadImage(property *image.Image, path string) error {
	if path == "" {
		return nil
	}

	path, err := expandPath(w.base, path)
	if err != nil {
		return err
	}
	icon, err := loadImage(path)
	if err != nil {
		return err
	}

	if w.flatten {
		*property = flattenImage(icon, w.color)
	} else {
		*property = icon
	}
	return nil
}

// SetImage updates the widget's icon.
func (w *ButtonWidget) SetImage(img image.Image) {
	w.icon = img
	if w.flatten {
		w.icon = flattenImage(w.icon, w.color)
	}
}

// Update renders the widget.
func (w *ButtonWidget) Update() error {
	size := int(w.dev.Pixels)
	margin := size / 18
	height := size - (margin * 2)
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	if w.label != "" {
		iconsize := int((float64(height) / 3.0) * 2.0)
		bounds := img.Bounds()

		if w.icon != nil {
			err := drawImage(img,
				w.icon,
				iconsize,
				image.Pt(-1, margin))

			if err != nil {
				return err
			}

			bounds.Min.Y += iconsize + margin
			bounds.Max.Y -= margin
		}

		drawString(img,
			bounds,
			ttfFont,
			w.label,
			w.dev.DPI,
			w.fontsize,
			w.color,
			image.Pt(-1, -1))
	} else if w.icon != nil {
		err := drawImage(img,
			w.icon,
			height,
			image.Pt(-1, -1))

		if err != nil {
			return err
		}
	}

	return w.render(w.dev, img)
}

// TriggerAction default action is to toggle the button image
func (w *ButtonWidget) TriggerAction(hold bool) {
	verbosef("toggle key for button %d", w.key)
	if w.disabled != nil {
		w.icon, w.disabled = w.disabled, w.icon
	}
	w.Update()
}
