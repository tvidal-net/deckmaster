package main

import "github.com/godbus/dbus/v5"

const (
	BecomeMonitorMethod = "org.freedesktop.DBus.Monitoring.BecomeMonitor"
)

var (
	dbusConnection *dbus.Conn
)

func DBusConnect() error {
	if dbusConnection != nil {
		return nil
	}

	dbusConnection, e := dbus.SessionBus()
	if e != nil {
		return e
	}

	if e := dbusConnection.Auth(nil); e != nil {
		dbusDisconnect()
		return e
	}

	if e := dbusConnection.Hello(); e != nil {
		dbusDisconnect()
		return e
	}
	return nil
}

func dbusDisconnect() {
	if dbusConnection != nil {
		_ = dbusConnection.Close()
	}
	dbusConnection = nil
}

func Monitor(rules ...string) <-chan *dbus.Signal {
	busObject := dbusConnection.BusObject()
	if call := busObject.Call(BecomeMonitorMethod, 0, rules, uint32(0)); call.Err != nil {
		return nil, call.Err
	}

	signals := make(chan *dbus.Signal)
	dbusConnection.Signal(signals)
	return signals
}

func CallDBus(object, path, method string, args ...interface{}) *dbus.Call {
	o := dbusConnection.Object(object, dbus.ObjectPath(path))
	return o.Call(method, 0, args...)
}
