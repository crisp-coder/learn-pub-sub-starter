package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	connectionStr := "amqp://guest:guest@localhost:5672/"
	connection, err := amqp.Dial(connectionStr)
	if err != nil {
		log.Println(err)
		return
	}
	defer connection.Close()

	ch, err := connection.Channel()
	if err != nil {
		log.Println(err)
		return
	}
	_, _, err = pubsub.DeclareAndBind(
		connection,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		pubsub.DURABLE,
	)
	if err != nil {
		log.Println(err)
		return
	}
	err = pubsub.SubscribeGob(
		connection,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		pubsub.DURABLE,
		handleLog,
		unmarshalLog,
	)

	gamelogic.PrintServerHelp()

	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}
		switch input[0] {
		case "pause":
			PublishPause(ch)
		case "resume":
			PublishResume(ch)
		case "quit":
			log.Println("exiting game...")
			return
		}
	}
}

func PublishPause(ch *amqp.Channel) {
	err := pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
		IsPaused: true,
	})
	if err != nil {
		log.Println(err)
		return
	}
}

func PublishResume(ch *amqp.Channel) {
	err := pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
		IsPaused: false,
	})
	if err != nil {
		log.Println(err)
		return
	}
}

func handleLog(gl routing.GameLog) pubsub.AckType {
	defer fmt.Print("> ")
	gamelogic.WriteLog(gl)
	return pubsub.ACK
}

func unmarshalLog(data []byte) (routing.GameLog, error) {
	gl := routing.GameLog{}
	decoder := gob.NewDecoder(bytes.NewReader(data))
	decoder.Decode(&gl)
	return gl, nil
}
