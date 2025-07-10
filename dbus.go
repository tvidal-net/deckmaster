package main

import (
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

const (
	dbusInterface   = "io.github.muesli.DeckMaster"
	dbusMonitorPath = "/Monitor"
	introInterface  = "org.freedesktop.DBus.Introspectable"
	intro           = `<node>
	<interface name="` + dbusInterface + `">
		<method name="ActiveWindowChanged">
			<arg direction="in" type="s" />
			<arg direction="in" type="s" />
		</method>
	</interface>` + introspect.IntrospectDataString + "<node>"
)

type WindowChannel struct {
	channel chan string
}

func (w *WindowChannel) ActiveWindowChanged(class, id string) *dbus.Error {
	w.channel <- class
	return nil
}

func MonitorActiveWindowChanged() <-chan string {
	w := WindowChannel{make(chan string)}
	cnn, _ := dbus.SessionBus()
	_ = cnn.Export(&w, dbusMonitorPath, dbusInterface)

	introspectable := introspect.Introspectable(intro)
	_ = cnn.Export(introspectable, dbusMonitorPath, introInterface)

	reply, e := cnn.RequestName(dbusInterface, dbus.NameFlagDoNotQueue)
	if e != nil {
		errorLog(e, "failed to request name on active window changed")
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		errorLogF("service '%s' already running", dbusInterface)
	}
	return w.channel
}

func CallDBus(object, path, method string, args ...interface{}) error {
	cnn, _ := dbus.SessionBus()

	o := cnn.Object(object, dbus.ObjectPath(path))
	return o.Call(method, 0, args...).Err
}
