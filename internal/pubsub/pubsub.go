package pubsub

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
    Durable SimpleQueueType = iota
    Transient
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
    valJSON, err := json.Marshal(val)
    if err != nil {
        return err
    }
    return ch.PublishWithContext(
        context.Background(),
        exchange,
        key,
        false,
        false,
        amqp.Publishing{
            ContentType: "application/json",
            Body:   valJSON,
        },
    )
}

func DeclareAndBind(
    conn *amqp.Connection,
    exchange,
    queueName,
    key string,
    queueType SimpleQueueType, 
) (*amqp.Channel, amqp.Queue, error) {
    var queue amqp.Queue
    connectionChannel, err := conn.Channel()
    if err != nil {
        return nil, queue, err
    }
    queue,err = connectionChannel.QueueDeclare(
        queueName,
        queueType == Durable,
        queueType == Transient,
        queueType == Transient,
        false,
        nil,
    )
    if err != nil {
        return nil, queue, err
    }
    if err = connectionChannel.QueueBind(queue.Name,key,exchange,false,nil); err != nil {
        return nil, queue, err
    }
    return connectionChannel, queue, nil
}
