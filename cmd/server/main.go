package main

import (
	"fmt"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
    const serverURL = "amqp://guest:guest@localhost:5672"
    conn, err := amqp.Dial(serverURL)
    if err != nil {
        panic(err)
    }
    defer conn.Close()
    fmt.Println("Successfully connected to rabbitmq server")
    sigChan, err := conn.Channel()
    if err != nil {
        panic(err)
    }
    pubsub.PublishJSON(sigChan,routing.ExchangePerilDirect,routing.PauseKey,routing.PlayingState{IsPaused: true})
}
