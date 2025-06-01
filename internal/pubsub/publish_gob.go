package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var encodedBody bytes.Buffer
	encoder := gob.NewEncoder(&encodedBody)
	err := encoder.Encode(val)
	if err != nil {
		return fmt.Errorf("error encoding value to gob: %v", err)
	}

	ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/gob",
		Body:        encodedBody.Bytes(),
	})

	return nil
}
