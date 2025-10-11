package pubsub

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	TRANSIENT SimpleQueueType = iota
	DURABLE
)

func DeclareAndBind(conn *amqp.Connection, exchange, queueName, key string, queueType SimpleQueueType) (*amqp.Channel, amqp.Queue, error) {

	ch, err := conn.Channel()
	if err != nil {
		log.Println(err)
		return nil, amqp.Queue{}, err
	}
	var durable bool
	var autoDelete bool
	var exclusive bool
	var noWait bool
	args := amqp.Table{}
	args["x-dead-letter-exchange"] = "peril_dlx"

	switch queueType {
	case DURABLE:
		durable = true
		autoDelete = false
		exclusive = false
		noWait = false
	case TRANSIENT:
		durable = false
		autoDelete = true
		exclusive = true
		noWait = false
	}
	q, err := ch.QueueDeclare(queueName, durable, autoDelete, exclusive, noWait, args)
	if err != nil {
		log.Println(err)
		return nil, amqp.Queue{}, err
	}
	err = ch.QueueBind(queueName, key, exchange, noWait, nil)
	if err != nil {
		log.Println(err)
		return nil, amqp.Queue{}, err
	}

	return ch, q, nil
}
