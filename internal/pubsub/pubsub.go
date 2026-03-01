package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	Durable SimpleQueueType = iota
	Transient
)

type SimpleAckType int

const (
	Ack int = iota
	NackRequeue
	NackDiscard
)

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
	queue, err = connectionChannel.QueueDeclare(
		queueName,
		queueType == Durable,
		queueType == Transient,
		queueType == Transient,
		false,
		amqp.Table{
			"x-dead-letter-exchange": "peril_dlx",
		},
	)
	if err != nil {
		return nil, queue, err
	}
	if err = connectionChannel.QueueBind(queue.Name, key, exchange, false, nil); err != nil {
		return nil, queue, err
	}
	return connectionChannel, queue, nil
}

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
			Body:        valJSON,
		},
	)
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) SimpleAckType,
) error {
	return subscribe(
		conn,
		exchange, queueName, key,
		queueType,
		handler,
		func(data []byte, buf *T) error {
			return json.Unmarshal(data, buf)
		},
	)
}

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(val); err != nil {
		return err
	}
	return ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/gob",
			Body:        buf.Bytes(),
		},
	)
}

func SubscribeGob[T any] (
	conn *amqp.Connection,
	exchange, queueName, key string,
	queueType SimpleQueueType,
	handler func(T) SimpleAckType,
) error {
    return subscribe(conn,
    exchange, queueName, key,
    queueType, handler,
    func (data []byte, buf *T) error  {
        bytesBuf := bytes.NewBuffer(data)
        dec := gob.NewDecoder(bytesBuf)
        return dec.Decode(buf)
    },
)

}

func subscribe[T any](
	conn *amqp.Connection,
	exchange, queueName, key string,
	queueType SimpleQueueType,
	handler func(T) SimpleAckType,
	unmarshaller func([]byte, *T) error,
) error {
	queueChan, _, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}
	deliveryChan, err := queueChan.Consume("", "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for d := range deliveryChan {
			var v T
			if err := unmarshaller(d.Body, &v); err != nil {
				fmt.Printf("error unmarshalling data: %v", err)
				d.Ack(false)
				continue
			}
			ackType := handler(v)
			switch ackType {
			case SimpleAckType(Ack):
				d.Ack(false)
			case SimpleAckType(NackRequeue):
				d.Nack(false, true)
			case SimpleAckType(NackDiscard):
				d.Nack(false, false)

			}
		}
	}()
	return nil
}
