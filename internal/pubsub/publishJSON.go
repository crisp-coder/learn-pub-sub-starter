package pubsub

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	data, err := json.Marshal(val)
	if err != nil {
		log.Println(err)
		return err
	}
	err = ch.Publish(exchange, key, false, false, amqp.Publishing{
		Body:        data,
		ContentType: "application/json",
	})
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
