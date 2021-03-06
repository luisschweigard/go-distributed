package coordinator

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/streadway/amqp"
	"go-distributed/src/distributed/dto"
	"go-distributed/src/distributed/qutils"
	"log"
)

const url = "amqp://guest:guest@localhost:5672"

type QueueListener struct {
	conn    *amqp.Connection
	ch      *amqp.Channel
	sources map[string]<-chan amqp.Delivery
	ea      *EventAggregator
}

func NewQueueListener(ea *EventAggregator) *QueueListener {
	ql := QueueListener{
		sources: make(map[string]<-chan amqp.Delivery),
		ea:      ea,
	}
	ql.conn, ql.ch = qutils.GetChannel(url)

	return &ql
}

func (ql *QueueListener) DiscoverSensors() {
	_ = ql.ch.ExchangeDeclare(
		qutils.SensorDiscoveryExchange,
		"fanout",
		false,
		false,
		false,
		false,
		nil,
	)

	_ = ql.ch.Publish(
		qutils.SensorDiscoveryExchange,
		"",
		false,
		false,
		amqp.Publishing{},
	)
}

func (ql *QueueListener) ListenForNewSource() {
	q := qutils.GetQueue("", ql.ch, true)
	_ = ql.ch.QueueBind(
		q.Name,
		"",
		"amq.fanout",
		false,
		nil,
	)

	msgs, _ := ql.ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)

	ql.DiscoverSensors()

	fmt.Println("Listening for new sources")
	for msg := range msgs {
		log.Println("New source discovered")
		ql.ea.PublishEvent("DataSourceDiscovered", string(msg.Body))
		sourceChan, _ := ql.ch.Consume(
			string(msg.Body),
			"",
			true,
			false,
			false,
			false,
			nil,
		)

		if ql.sources[string(msg.Body)] == nil {
			ql.sources[string(msg.Body)] = sourceChan

			go ql.AddListener(sourceChan)
		}
	}
}

func (ql *QueueListener) AddListener(msgs <-chan amqp.Delivery) {
	for msg := range msgs {
		r := bytes.NewReader(msg.Body)
		d := gob.NewDecoder(r)

		sd := new(dto.SensorMessage)
		_ = d.Decode(sd)

		fmt.Printf("Received message: %v\n", sd)

		ed := EventData{
			Name:      sd.Name,
			Value:     sd.Value,
			Timestamp: sd.Timestamp,
		}

		ql.ea.PublishEvent("MessageReceived_"+msg.RoutingKey, ed)
	}
}
