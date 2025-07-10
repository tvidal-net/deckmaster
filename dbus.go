package main

import "github.com/godbus/dbus/v5"

const (
	BecomeMonitorMethod = "org.freedesktop.DBus.Monitoring.BecomeMonitor"
)

func CallDBus(object, path, method string, args ...interface{}) error {
	cnn, _ := dbus.SessionBus()

	o := cnn.Object(object, dbus.ObjectPath(path))
	return o.Call(method, 0, args...).Err
}
