package pubsub

import (
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) Acktype,
) error {
	ch, queue, err := DeclareAndBind(conn, exchange, queueName, key, SimpleQueueType(simpleQueueType))
	if err != nil {
		return fmt.Errorf("error checking and binding queue: %v", err)
	}

	err = ch.Qos(10, 0, false)
	if err != nil {
		return fmt.Errorf("could not set QoS: %v", err)
	}

	deliveries, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("error consuming and delivering: %v", err)
	}

	go func(ch <-chan amqp.Delivery) {
		for delivery := range ch {
			var body T
			delerr := json.Unmarshal(delivery.Body, &body)
			if delerr != nil {
				log.Printf("\nerror unmarshalling delivery: %v", delerr)
				continue
			}
			switch handler(body) {
			case Ack:
				delivery.Ack(false)
				fmt.Println("Ack")
			case NackDiscard:
				delivery.Nack(false, false)
				fmt.Println("NackDiscard")
			case NackRequeue:
				delivery.Nack(false, true)
				fmt.Println("NackRequeue")
			}

		}
	}(deliveries)

	return nil
}
