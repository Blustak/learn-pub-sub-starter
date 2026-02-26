package main

import (
	"fmt"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
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
    _, _, err = pubsub.DeclareAndBind(conn,
        "peril_topic",
        "game_logs",
        "game_logs.*",
        pubsub.Durable,
    )
    if err != nil {
        panic(err)
    }
    sigChan, err := conn.Channel()
    if err != nil {
        panic(err)
    }
    gamelogic.PrintServerHelp()
    running := true
    for running {
        words := gamelogic.GetInput()
        for _,w := range words {
            switch w {
            case "pause":
                fmt.Println("Pausing game...")
                pubsub.PublishJSON(sigChan,routing.ExchangePerilDirect,routing.PauseKey,routing.PlayingState{IsPaused: true})
            case "resume":
                fmt.Println("Resuming game...")
                pubsub.PublishJSON(sigChan,routing.ExchangePerilDirect,routing.PauseKey,routing.PlayingState{IsPaused: false})
            case "quit":
                fmt.Println("Exiting...")
                running = false
            default:
                fmt.Printf("Unkown command %s\n", w)
            }
        }
    }
}
