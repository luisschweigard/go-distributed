package coordinator

import (
	"time"
)

type EventRaiser interface {
	AddListener(eventName string, f func(interface{}))
}

type EventAggregator struct {
	listeners map[string][]func(interface{})
}

type EventData struct {
	Name      string
	Value     float64
	Timestamp time.Time
}

func NewEventAggregator() *EventAggregator {
	return &EventAggregator{listeners: make(map[string][]func(interface{}))}
}

func (ea *EventAggregator) AddListener(name string, f func(interface{})) {
	ea.listeners[name] = append(ea.listeners[name], f)
}

func (ea *EventAggregator) PublishEvent(name string, eventData interface{}) {
	if ea.listeners[name] != nil {
		for _, cb := range ea.listeners[name] {
			cb(eventData)
		}
	}
}