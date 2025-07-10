package main

import (
	"github.com/tvidal-net/pulseaudio"
	"strings"
	"time"
)

const (
	SinkChanged = iota
	SourceChanged
	SinkMuteChanged
	SourceMuteChanged
)

type ChangeType uint8

type PulseAudio struct {
	client        pulseaudio.Client
	currentSink   pulseaudio.Sink
	currentSource pulseaudio.Source
	updates       chan ChangeType
}

func getSink(name string, client *pulseaudio.Client) (*pulseaudio.Sink, error) {
	sinks, err := client.Sinks()
	if err != nil {
		return nil, err
	}
	for _, sink := range sinks {
		if sink.Name == name {
			return &sink, nil
		}
	}
	return nil, &pulseaudio.Error{Cmd: "getSink", Code: 3}
}

func getSource(name string, client *pulseaudio.Client) (*pulseaudio.Source, error) {
	sources, err := client.Sources()
	if err != nil {
		return nil, err
	}
	for _, source := range sources {
		if source.Name == name {
			return &source, nil
		}
	}
	return nil, &pulseaudio.Error{Cmd: "getSource", Code: 3}
}

func NewPulseAudio() (*PulseAudio, error) {
	client, err := pulseaudio.NewClient()
	if err != nil {
		return nil, err
	}

	serverInfo, err := client.ServerInfo()
	if err != nil {
		client.Close()
		return nil, err
	}

	defaultSink, err := getSink(serverInfo.DefaultSink, client)
	if err != nil {
		client.Close()
		return nil, err
	}

	defaultSource, err := getSource(serverInfo.DefaultSource, client)
	if err != nil {
		client.Close()
		return nil, err
	}
	pulseAudio := &PulseAudio{
		*client,
		*defaultSink,
		*defaultSource,
		make(chan ChangeType),
	}
	return pulseAudio, nil
}

func (c *PulseAudio) Updates() <-chan ChangeType {
	return c.updates
}

func (c *PulseAudio) Start() {
	var pulseAudioUpdates <-chan struct{}
	for {
		clientUpdates, e := c.client.Updates()
		if e != nil {
			errorLog(e, "failed to subscribe to PulseAudio updates")
			time.Sleep(time.Second)
		} else {
			pulseAudioUpdates = clientUpdates
			break
		}
	}
	for {
		_ = <-pulseAudioUpdates
		serverInfo, e := c.client.ServerInfo()
		if e != nil {
			errorLog(e, "failed to get PulseAudio server info")
			continue
		}

		defaultSink, e := getSink(serverInfo.DefaultSink, &c.client)
		if e != nil {
			errorLog(e, "failed to get PulseAudio sinks")
			continue
		}
		if defaultSink.Name != c.CurrentSinkName() {
			c.currentSink = *defaultSink
			c.updates <- SinkChanged
		}
		if defaultSink.Muted != c.currentSink.Muted {
			c.currentSink = *defaultSink
			c.updates <- SinkMuteChanged
		}

		defaultSource, e := getSource(serverInfo.DefaultSource, &c.client)
		if e != nil {
			errorLog(e, "failed to get PulseAudio sources")
			continue
		}
		if defaultSource.Name != c.CurrentSourceName() {
			c.currentSource = *defaultSource
			c.updates <- SourceChanged
		}
		if defaultSource.Muted != c.currentSource.Muted {
			c.currentSource = *defaultSource
			c.updates <- SourceMuteChanged
		}
	}
}

func (c *PulseAudio) Muted(isSinkStream bool) bool {
	if isSinkStream {
		return c.currentSink.Muted
	} else {
		return c.currentSource.Muted
	}
}

func (c *PulseAudio) ToggleMute(isSinkStream bool) error {
	if isSinkStream {
		return c.client.SetSinkMute(!c.currentSink.Muted, c.CurrentSinkName())
	} else {
		return c.client.SetSourceMute(!c.currentSource.Muted, c.CurrentSourceName())
	}
}

func (c *PulseAudio) CurrentSinkName() string {
	return c.currentSink.Name
}

func (c *PulseAudio) SetSink(partialName string) error {
	verboseLog("currentSink: %s", c.CurrentSinkName())
	sinks, err := c.client.Sinks()
	if err != nil {
		return err
	}
	for _, sink := range sinks {
		sinkName := sink.Name
		if sink.Name != c.CurrentSinkName() && strings.Contains(sinkName, partialName) {
			verboseLog("setSink \"%s\"=%s", partialName, sinkName)
			return c.client.SetDefaultSink(sinkName)
		}
	}
	return nil
}

func (c *PulseAudio) CurrentSourceName() string {
	return c.currentSource.Name
}

func (c *PulseAudio) SetSource(partialName string) error {
	verboseLog("currentSource: %s", c.CurrentSourceName())
	sources, err := c.client.Sources()
	if err != nil {
		return err
	}
	for _, source := range sources {
		if source.MonitorSourceName == "" {
			sourceName := source.Name
			if source.Name != c.CurrentSourceName() && strings.Contains(sourceName, partialName) {
				verboseLog("setSource \"%s\"=%s", partialName, sourceName)
				return c.client.SetDefaultSource(sourceName)
			}
		}
	}
	return nil
}

func (c *PulseAudio) Close() {
	c.client.Close()
}
