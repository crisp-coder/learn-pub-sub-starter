package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

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
		handlerMove(ch, gamestate))

	warqueue := routing.WarRecognitionsPrefix
	pubsub.SubscribeJSON(
		connection,
		routing.ExchangePerilTopic,
		warqueue,
		routing.WarRecognitionsPrefix+".*",
		pubsub.DURABLE,
		handlerWar(ch, gamestate),
	)

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
			if len(input) != 2 {
				fmt.Println("invalid input")
				continue
			}

			times, err := strconv.Atoi(input[1])
			if err != nil {
				fmt.Println("invalid input")
				continue
			}
			for i := 0; i < times; i++ {
				msg := gamelogic.GetMaliciousLog()
				pubsub.PublishGob(ch, routing.ExchangePerilTopic, routing.GameLogSlug+"."+username, routing.GameLog{
					CurrentTime: time.Now(),
					Message:     msg,
					Username:    username,
				})
			}

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

func handlerMove(ch *amqp.Channel, gs *gamelogic.GameState) func(am gamelogic.ArmyMove) pubsub.AckType {
	handler := func(am gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		outcome := gs.HandleMove(am)
		switch outcome {
		case gamelogic.MoveOutcomeSafe:
			return pubsub.ACK
		case gamelogic.MoveOutcomeMakeWar:
			pubsub.PublishJSON(
				ch,
				routing.ExchangePerilTopic,
				routing.WarRecognitionsPrefix+"."+gs.Player.Username,
				gamelogic.RecognitionOfWar{
					Attacker: am.Player,
					Defender: gs.Player,
				},
			)
			return pubsub.ACK
		default:
			return pubsub.NACKDISCARD
		}
	}
	return handler
}

func handlerWar(ch *amqp.Channel, gs *gamelogic.GameState) func(war gamelogic.RecognitionOfWar) pubsub.AckType {

	handler := func(war gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		outcome, winner, loser := gs.HandleWar(war)

		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NACKREQUEUE
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NACKDISCARD
		case gamelogic.WarOutcomeOpponentWon:
			msg := fmt.Sprintf("%v won a war against %v", winner, loser)
			pubsub.PublishGob(ch, routing.ExchangePerilTopic, routing.GameLogSlug+"."+war.Attacker.Username, msg)
			return pubsub.ACK
		case gamelogic.WarOutcomeYouWon:
			msg := fmt.Sprintf("%v won a war against %v", winner, loser)
			pubsub.PublishGob(ch, routing.ExchangePerilTopic, routing.GameLogSlug+"."+war.Attacker.Username, msg)
			return pubsub.ACK
		case gamelogic.WarOutcomeDraw:
			msg := fmt.Sprintf("A war between %v and %v resulted in a draw", winner, loser)
			pubsub.PublishGob(ch, routing.ExchangePerilTopic, routing.GameLogSlug+"."+war.Attacker.Username, msg)
			return pubsub.ACK
		default:
			log.Println("unrecognized war outcome")
			return pubsub.NACKDISCARD
		}

	}

	return handler
}
