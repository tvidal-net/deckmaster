package main

import (
	"fmt"
	"github.com/lawl/pulseaudio"
	"os"
	"time"
)

type PulseAudio struct {
	client        pulseaudio.Client
	defaultSink   pulseaudio.Sink
	defaultSource pulseaudio.Source
	updates       chan ChangeType
}

const (
	SinkChanged = iota
	SourceChanged
	SinkMuteChanged
	SourceMuteChanged
)

type ChangeType uint8

func paError(err error) {
	fmt.Fprintln(os.Stderr, "pulseaudio client error:", err)
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
	return nil, nil
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
	return nil, nil
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
	return &PulseAudio{
		*client,
		*defaultSink,
		*defaultSource,
		make(chan ChangeType),
	}, nil
}

func (c *PulseAudio) Updates() chan ChangeType {
	return c.updates
}

func (c *PulseAudio) Start() {
	var pulseAudioUpdates <-chan struct{}
	for {
		clientUpdates, err := c.client.Updates()
		if err != nil {
			paError(err)
			time.Sleep(time.Second)
		} else {
			pulseAudioUpdates = clientUpdates
			break
		}
	}
	for {
		select {
		case _ = <-pulseAudioUpdates:
			serverInfo, err := c.client.ServerInfo()
			if err != nil {
				errorf(err)
				continue
			}
			defaultSink, err := getSink(serverInfo.DefaultSink, &c.client)
			if err != nil {
				errorf(err)
			} else {
				if serverInfo.DefaultSink != c.defaultSink.Name {
					c.defaultSink = *defaultSink
					c.updates <- SinkChanged
				}
				if c.defaultSink.Muted != defaultSink.Muted {
					c.defaultSink = *defaultSink
					c.updates <- SinkMuteChanged
				}
			}
			defaultSource, err := getSource(serverInfo.DefaultSource, &c.client)
			if err != nil {
				errorf(err)
			} else {
				if serverInfo.DefaultSource != c.defaultSource.Name {
					c.defaultSource = *defaultSource
					c.updates <- SourceChanged
				}
				if c.defaultSource.Muted != defaultSource.Muted {
					c.defaultSource = *defaultSource
					c.updates <- SourceMuteChanged
				}
			}
		}
	}
}

func (c *PulseAudio) Close() {
	c.client.Close()
}
