package pubsub

import (
	"bytes"
	"encoding/gob"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	buf := bytes.NewBuffer(make([]byte, 0))
	encoder := gob.NewEncoder(buf)
	err := encoder.Encode(val)
	if err != nil {
		log.Println(err)
		return err
	}
	data := buf.Bytes()
	err = ch.Publish(exchange, key, false, false, amqp.Publishing{
		Body:        data,
		ContentType: "application/gob",
	})
	if err != nil {
		log.Println(err)
		return err
	}
	log.Printf("Publishing %v\n", val)
	return nil

}
