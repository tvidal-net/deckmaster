package main

import (
	"github.com/godbus/dbus/v5"
)

const (
	BecomeMonitorMethod = "org.freedesktop.DBus.Monitoring.BecomeMonitor"
)

var (
	dbusConnection *dbus.Conn
)

type DBusMonitorChannel struct {
	monitorChannel chan *dbus.Signal
}

type DBusMonitor interface {
	Start()
	Close()
	Channel() chan *dbus.Signal
}

func newDBusMonitorConnection() (*dbus.Conn, error) {
	cnn, err := dbus.SessionBusPrivate()
	if err != nil {
		return nil, err
	}
	err = cnn.Auth(nil)
	if err != nil {
		_ = cnn.Close()
		return nil, err
	}
	err = cnn.Hello()
	if err != nil {
		_ = cnn.Close()
		return nil, err
	}
	return cnn, nil
}

func NewDBusMonitor(rules ...string) (DBusMonitor, error) {
	if dbusConnection == nil {
		cnn, err := newDBusMonitorConnection()
		if err != nil {
			panic(err)
		}
		dbusConnection = cnn
	}
	res := dbusConnection.BusObject().Call(BecomeMonitorMethod, 0, rules, uint32(0))
	if res.Err != nil {
		return nil, res.Err
	}
	return &DBusMonitorChannel{make(chan *dbus.Signal)}, nil
}

func (c *DBusMonitorChannel) Channel() chan *dbus.Signal {
	return c.monitorChannel
}

func (c *DBusMonitorChannel) Start() {
	dbusConnection.Signal(c.monitorChannel)
}

func (c *DBusMonitorChannel) Close() {
	dbusConnection.RemoveSignal(c.monitorChannel)
}
