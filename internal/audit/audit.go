package audit

import (
	"context"
	"errors"
)

type Event struct {
	Timestamp int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

type Observer interface {
	Notify(context.Context, Event) error
}

type Publisher struct {
	observers []Observer
}

func NewPublisher(observers ...Observer) *Publisher {
	return &Publisher{observers: append([]Observer(nil), observers...)}
}

func (p *Publisher) Notify(ctx context.Context, event Event) error {
	var err error
	for _, observer := range p.observers {
		err = errors.Join(err, observer.Notify(ctx, event))
	}
	return err
}
