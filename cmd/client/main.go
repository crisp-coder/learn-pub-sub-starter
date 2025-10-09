package main

import (
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

	_, err = connection.Channel()
	if err != nil {
		log.Println(err)
		return
	}

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Println(err)
		return
	}
	_, _, err = pubsub.DeclareAndBind(
		connection,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+username,
		routing.PauseKey,
		pubsub.TRANSIENT,
	)
	if err != nil {
		log.Println(err)
		return
	}

	gamestate := gamelogic.NewGameState(username)

	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}

		switch input[0] {
		case "spawn":
			err = gamestate.CommandSpawn(input)
			if err != nil {
				log.Println(err)
			}
		case "move":
			_, err := gamestate.CommandMove(input)
			if err != nil {
				log.Println(err)
			}
			log.Println("move successful")
		case "status":
			gamestate.CommandStatus()
		case "spam":
			log.Println("spamming not allowed yet")
		case "help":
			gamelogic.PrintClientHelp()
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			log.Println("unkown command")
		}
	}
}
