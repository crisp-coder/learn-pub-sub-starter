package main

import (
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

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Println(err)
		return
	}
	queueName := routing.PauseKey + "." + username
	_, _, err = pubsub.DeclareAndBind(
		connection,
		routing.ExchangePerilDirect,
		queueName,
		routing.PauseKey,
		pubsub.TRANSIENT,
	)
	if err != nil {
		log.Println(err)
		return
	}

	gamestate := gamelogic.NewGameState(username)
	pubsub.SubscribeJSON(
		connection,
		routing.ExchangePerilDirect,
		queueName,
		routing.PauseKey,
		pubsub.TRANSIENT,
		handlerPause(gamestate))

	moveQueue := routing.ArmyMovesPrefix + "." + username
	pubsub.SubscribeJSON(
		connection,
		routing.ExchangePerilTopic,
		moveQueue,
		routing.ArmyMovesPrefix+".*",
		pubsub.TRANSIENT,
		handlerMove(gamestate))

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
			armyMove, err := gamestate.CommandMove(input)
			if err != nil {
				log.Println(err)
			}
			pubsub.PublishJSON(ch, routing.ExchangePerilTopic, moveQueue, armyMove)
			log.Println("Published move message")
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

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	handler := func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.ACK
	}

	return handler
}

func handlerMove(gs *gamelogic.GameState) func(am gamelogic.ArmyMove) pubsub.AckType {
	handler := func(am gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		outcome := gs.HandleMove(am)
		if outcome == gamelogic.MoveOutComeSafe || outcome == gamelogic.MoveOutcomeMakeWar {
			return pubsub.ACK
		}
		return pubsub.NACKDISCARD
	}
	return handler
}
