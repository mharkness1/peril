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

const connectionUrl string = "amqp://guest:guest@localhost:5672/"

func main() {
	fmt.Println("Starting Peril client...")

	connection, err := amqp.Dial(connectionUrl)
	if err != nil {
		log.Fatalf("Error creating rabbitmq connection: %v", err)
	}
	defer connection.Close()
	fmt.Println("Connection to Peril Server succeeded...")

	publishCh, err := connection.Channel()
	if err != nil {
		log.Fatalf("Couldn't created channel: %v", err)
	}

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Error getting client username: %v", err)
	}

	_, queue, err := pubsub.DeclareAndBind(
		connection,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+username,
		routing.PauseKey,
		pubsub.SimpleQueueTransient,
	)
	if err != nil {
		log.Fatalf("Couldn't subscribe to pause: %v", err)
	}

	fmt.Printf("Queue %v created and bound\n", queue.Name)

	gs := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(connection, routing.ExchangePerilDirect, "pause."+username, routing.PauseKey, pubsub.SimpleQueueTransient, handlerPause(gs))
	if err != nil {
		log.Fatalf("Failed to subscribe to pause queue: %v", err)
	}

	err = pubsub.SubscribeJSON(connection, routing.ExchangePerilTopic, "army_moves."+username, "army_moves.*", pubsub.SimpleQueueTransient, handlerMove(gs, publishCh))
	if err != nil {
		log.Fatalf("Failed to subscribe to move queue: %v", err)
	}

	err = pubsub.SubscribeJSON(connection, routing.ExchangePerilTopic, routing.WarRecognitionsPrefix, routing.WarRecognitionsPrefix+".*", pubsub.SimpleQueueDurable, handlerWar(gs, publishCh))
	if err != nil {
		log.Fatalf("could not subscribe to war declarations: %v", err)
	}

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "spawn":
			err = gs.CommandSpawn(words)
			if err != nil {
				fmt.Printf("%v", err)
			}
		case "move":
			mv, err := gs.CommandMove(words)
			if err != nil {
				fmt.Printf("%v", err)
			}
			err = pubsub.PublishJSON(publishCh, routing.ExchangePerilTopic, routing.ArmyMovesPrefix+"."+mv.Player.Username, mv)
			if err != nil {
				fmt.Printf("error publishing move: %v\n", err)
				continue
			}
			fmt.Printf("Moved %v units to %s\n", len(mv.Units), mv.ToLocation)
		case "status":
			gs.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			if len(words) < 2 {
				fmt.Printf("error, spam requires a argument")
				continue
			}
			num, err := strconv.Atoi(words[1])
			if err != nil {
				fmt.Printf("error, argument needs to be an int")
				continue
			}
			for i := 0; i < num; i++ {
				message := gamelogic.GetMaliciousLog()
				err = publishGameLog(publishCh, username, message)
				if err != nil {
					fmt.Printf("error publishing game log: %v", err)
					continue
				}
			}

		case "quit":
			log.Println("Goodbye!")
			return
		default:
			fmt.Println("Unknown command")
		}
	}
}

func publishGameLog(publishCh *amqp.Channel, username, msg string) error {
	return pubsub.PublishGob(
		publishCh,
		routing.ExchangePerilTopic,
		routing.GameLogSlug+"."+username,
		routing.GameLog{
			Username:    username,
			CurrentTime: time.Now(),
			Message:     msg,
		},
	)
}
