package coordinator

import (
	"bytes"
	"encoding/gob"
	"github.com/streadway/amqp"
	"go-distributed/src/distributed/dto"
	"go-distributed/src/distributed/qutils"
)

type WebappConsumer struct {
	er      EventRaiser
	conn    *amqp.Connection
	ch      *amqp.Channel
	sources []string
}

func NewWebappConsumer(er EventRaiser) *WebappConsumer {
	wc := WebappConsumer{
		er: er,
	}

	wc.conn, wc.ch = qutils.GetChannel(url)
	qutils.GetQueue(qutils.PersistReadingsQueue, wc.ch, false)

	go wc.ListenForDiscoveryRequests()

	wc.er.AddListener("DataSourceDiscovered", func(eventData interface{}) {
		wc.SubscribeToDataEvent(eventData.(string))
	})

	_ = wc.ch.ExchangeDeclare(
		qutils.WebappSourceExchange,
		"fanout",
		false,
		false,
		false,
		false,
		nil,
	)

	_ = wc.ch.ExchangeDeclare(
		qutils.WebappReadingsExchange,
		"fanout",
		false,
		false,
		false,
		false,
		nil,
	)

	return &wc
}

func (wc *WebappConsumer) ListenForDiscoveryRequests() {
	q := qutils.GetQueue(qutils.WebappDiscoveryQueue, wc.ch, false)
	msgs, _ := wc.ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)

	for range msgs {
		for _, src := range wc.sources {
			wc.SendMessageSource(src)
		}
	}
}

func (wc *WebappConsumer) SendMessageSource(src string) {
	_ = wc.ch.Publish(
		qutils.WebappSourceExchange,
		"",
		false,
		false,
		amqp.Publishing{Body: []byte(src)},
	)
}

func (wc *WebappConsumer) SubscribeToDataEvent(eventName string) {
	for _, v := range wc.sources {
		if v == eventName {
			return
		}
	}

	wc.sources = append(wc.sources, eventName)
	wc.SendMessageSource(eventName)

	wc.er.AddListener("MessageReceived_"+eventName, func(eventData interface{}) {
		ed := eventData.(EventData)
		sm := dto.SensorMessage{
			Name:      ed.Name,
			Value:     ed.Value,
			Timestamp: ed.Timestamp,
		}

		buf := new(bytes.Buffer)
		enc := gob.NewEncoder(buf)
		_ = enc.Encode(sm)

		msg := amqp.Publishing{
			Body: buf.Bytes(),
		}

		_ = wc.ch.Publish(
			qutils.WebappReadingsExchange,
			"",
			false,
			false,
			msg,
		)
	})
}
