package audit

import (
	"context"
	"errors"
)

// Event содержит время обновления метрик, их имена и IP адрес отправителя
type Event struct {
	Timestamp int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

// Observer принимает события аудита
type Observer interface {
	// Notify передаёт событие получателю аудита
	Notify(context.Context, Event) error
}

// Publisher рассылает события зарегистрированным получателям
type Publisher struct {
	observers []Observer
}

// NewPublisher создаёт отправителя событий для указанных получателей аудита
func NewPublisher(observers ...Observer) *Publisher {
	return &Publisher{observers: append([]Observer(nil), observers...)}
}

// Notify отправляет событие всем получателям и объединяет их ошибки
func (p *Publisher) Notify(ctx context.Context, event Event) error {
	var err error
	for _, observer := range p.observers {
		err = errors.Join(err, observer.Notify(ctx, event))
	}
	return err
}
