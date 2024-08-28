package main

// RecentWindowWidget is a widget displaying a recently activated window.
type RecentWindowWidget struct {
	*ButtonWidget

	window    uint8
	showTitle bool

	lastID uint32
}

// NewRecentWindowWidget returns a new RecentWindowWidget.
func NewRecentWindowWidget(bw *BaseWidget, opts WidgetConfig) (*RecentWindowWidget, error) {
	var window int64
	if err := ConfigValue(opts.Config["window"], &window); err != nil {
		return nil, err
	}
	var showTitle bool
	_ = ConfigValue(opts.Config["showTitle"], &showTitle)

	widget, err := NewButtonWidget(bw, opts)
	if err != nil {
		return nil, err
	}

	return &RecentWindowWidget{
		ButtonWidget: widget,
		window:       uint8(window),
		showTitle:    showTitle,
	}, nil
}
