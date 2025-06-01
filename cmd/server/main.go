package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

const connectionUrl string = "amqp://guest:guest@localhost:5672/"

func main() {
	fmt.Println("Starting Peril server...")
	connection, err := amqp.Dial(connectionUrl)
	if err != nil {
		log.Fatalf("Error creating rabbitmq connection: %v", err)
	}
	defer connection.Close()
	fmt.Println("Connection to Peril Server succeeded...")

	connChan, err := connection.Channel()
	if err != nil {
		log.Fatalf("Error creating connection channel: %v", err)
	}

	err = pubsub.SubscribeGob(
		connection,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		pubsub.SimpleQueueDurable,
		func(log routing.GameLog) pubsub.Acktype {
			defer fmt.Print("\n>")

			err := gamelogic.WriteLog(log)
			if err != nil {
				fmt.Printf("\nerror writing log: %v", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		},
	)
	if err != nil {
		log.Fatalf("Couldn't subscribe to game logs: %v", err)
	}

	gamelogic.PrintServerHelp()

	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "pause":
			fmt.Println("Publishing pause message")
			err = pubsub.PublishJSON(
				connChan,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: true,
				},
			)
			if err != nil {
				log.Fatalf("Error publishing playing state to channel: %v", err)
			}
		case "resume":
			fmt.Println("Publishing resume message")
			err = pubsub.PublishJSON(
				connChan,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: false,
				},
			)
			if err != nil {
				log.Fatalf("Error publishing playing state to channel: %v", err)
			}
		case "quit":
			log.Println("Goodbye!")
			return
		default:
			fmt.Println("Unknown command")
		}
	}
}
