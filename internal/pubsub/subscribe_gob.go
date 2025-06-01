package pubsub

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func SubscribeGob[T any](
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
	deliveries, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("error consuming and delivering: %v", err)
	}

	go func(ch <-chan amqp.Delivery) {
		for delivery := range ch {
			buffer := bytes.NewBuffer(delivery.Body)
			decoder := gob.NewDecoder(buffer)
			var body T
			delerr := decoder.Decode(&body)
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
