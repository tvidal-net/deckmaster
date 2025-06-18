package main

import (
	"github.com/godbus/dbus"
)

func NewDBusMonitorConnection(name string) (*dbus.Conn, error) {
	// get a private connection
	cnn, err := dbus.SessionBusPrivate()
	if err != nil {
		return nil, err
	}

	// authenticate
	err = cnn.Auth(nil)
	if err != nil {
		_ = cnn.Close()
		return nil, err
	}

	// hello
	err = cnn.Hello()
	if err != nil {
		_ = cnn.Close()
		return nil, err
	}

	// request new name to workaround a bug in godbus
	_, err = cnn.RequestName(name, 0)
	if err != nil {
		_ = cnn.Close()
		return nil, err
	}
	return cnn, nil
}

func BecomeDBusMonitor(cnn *dbus.Conn, rules ...string) {
	call := cnn.BusObject().Call("org.freedesktop.DBus.Monitoring.BecomeMonitor", 0, rules, uint32(0))
	if call.Err != nil {
		errorf(call.Err)
	}
}
