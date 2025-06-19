package main

import (
	"github.com/godbus/dbus/v5"
	"strings"
)

const (
	KMixInterface         = "org.kde.kmix"
	KMixerInterfaceRule   = "interface=org.kde.KMix.Mixer"
	CaptureDevicesPath    = "/Mixers/PulseAudio__Capture_Devices_1"
	PlaybackDevicesPath   = "/Mixers/PulseAudio__Playback_Devices_1"
	MasterControlProperty = "org.kde.KMix.Mixer.masterControl"
	MuteProperty          = "org.kde.KMix.Control.mute"
	ControlChangedSignal  = "controlChanged"
)

type KMixerConnection struct {
	dbus *dbus.Conn
}

type KMixerObject struct {
	connection *KMixerConnection
	busObject  dbus.BusObject
}

type KMixerControl interface {
	GetMasterDevice() (KMixerDevice, error)
}

type KMixerDevice interface {
	GetMute() (bool, error)
	SetMute(bool) error
}

func isControlChanged(signal *dbus.Signal) bool {
	return signal != nil && strings.HasSuffix(signal.Name, "ControlChanged")
}

func NewKMixerConnection() KMixerConnection {
	cnn, err := newDBusConnection()
	if err != nil {
		panic(err)
	}
	return KMixerConnection{cnn}
}

func (cnn *KMixerConnection) Object(path dbus.ObjectPath) *KMixerObject {
	return &KMixerObject{
		connection: cnn,
		busObject:  cnn.dbus.Object(KMixInterface, path),
	}
}

func (cnn *KMixerConnection) GetCaptureControl() KMixerControl {
	return cnn.Object(CaptureDevicesPath)
}

func (cnn *KMixerConnection) GetPlaybackControl() KMixerControl {
	return cnn.Object(PlaybackDevicesPath)
}

func (cnn *KMixerConnection) GetCaptureDevice() (KMixerDevice, error) {
	control := cnn.GetCaptureControl()
	return control.GetMasterDevice()
}

func (cnn *KMixerConnection) GetPlaybackDevice() (KMixerDevice, error) {
	control := cnn.GetPlaybackControl()
	return control.GetMasterDevice()
}

func (cnn *KMixerConnection) isMicMuted() (bool, error) {
	device, err := cnn.GetCaptureDevice()
	if err != nil {
		return false, err
	}
	return device.GetMute()
}

func (cnn *KMixerConnection) ToggleMicMute() error {
	device, err := cnn.GetCaptureDevice()
	if err != nil {
		return err
	}
	mute, err := cnn.isMicMuted()
	if err != nil {
		return err
	}
	return device.SetMute(!mute)
}

func (cnn *KMixerConnection) isPlaybackMuted() (bool, error) {
	device, err := cnn.GetPlaybackDevice()
	if err != nil {
		return false, err
	}
	return device.GetMute()
}

func (cnn *KMixerConnection) TogglePlaybackMute() error {
	device, err := cnn.GetPlaybackDevice()
	if err != nil {
		return err
	}
	mute, err := device.GetMute()
	if err != nil {
		return err
	}
	return device.SetMute(!mute)
}

func (cnn *KMixerConnection) Close() {
	err := cnn.dbus.Close()
	if err != nil {
		panic(err)
	}
}

func (o *KMixerObject) GetMasterDevice() (KMixerDevice, error) {
	val, err := o.busObject.GetProperty(MasterControlProperty)
	if err != nil {
		return nil, err
	}
	path := dbus.ObjectPath(val.String())
	return o.connection.Object(path), nil
}

func (o *KMixerObject) GetMute() (bool, error) {
	val, err := o.busObject.GetProperty(MuteProperty)
	if err != nil {
		return false, err
	}
	return val.Value().(bool), nil
}

func (o *KMixerObject) SetMute(value bool) error {
	return o.busObject.SetProperty(MuteProperty, value)
}
