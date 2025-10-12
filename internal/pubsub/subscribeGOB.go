package pubsub

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
) error {
	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		log.Println(err)
		return err
	}
	consumerChannel, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		log.Println(err)
		return err
	}
	ch.Qos(10, 0, false)
	go func(consChan <-chan amqp.Delivery) {
		for delivery := range consChan {
			msg, err := unmarshaller(delivery.Body)
			if err != nil {
				log.Println(err)
				continue
			}
			ack := handler(msg)
			switch ack {
			case ACK:
				log.Println("Sending ACK")
				err = delivery.Ack(false)
				if err != nil {
					log.Println(err)
					continue
				}
			case NACKREQUEUE:
				log.Println("Sending NACKREQUEUE")
				err = delivery.Nack(false, true)
				if err != nil {
					log.Println(err)
					continue
				}
			case NACKDISCARD:
				log.Println("Sending NACKDISCARD")
				err = delivery.Nack(false, false)
				if err != nil {
					log.Println(err)
					continue
				}
			}
		}
	}(consumerChannel)
	return nil
}
