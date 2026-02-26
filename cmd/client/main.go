package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
    const rabbitmqServerUrl = "amqp://guest:guest@localhost:5672"
    conn, err := amqp.Dial(rabbitmqServerUrl)
    if err != nil {
        panic(err)
    }
    defer conn.Close()
    rabbitmqChannel, err := conn.Channel()
    if err != nil {
        panic(err)
    }
    defer rabbitmqChannel.Close()

    welcomeRes, err := gamelogic.ClientWelcome()
    if err != nil {
        panic(err)
    }
    psChan,psQueue,err := pubsub.DeclareAndBind(
        conn,
        routing.ExchangePerilDirect,
        routing.PauseKey + "." + welcomeRes,
        routing.PauseKey,
        pubsub.Transient,
    )
    if err != nil {
        panic(err)
    }
    defer psChan.Close()
    fmt.Printf("Queue %s created. \n", psQueue.Name)

    //Wait for ctrl+c
    stdinChan := make(chan os.Signal, 1)
    signal.Notify(stdinChan,os.Interrupt)
    <-stdinChan
    fmt.Println("Shutting down...")
}
