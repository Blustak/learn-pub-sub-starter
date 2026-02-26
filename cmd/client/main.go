package main

import (
	"fmt"

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
    gameState := gamelogic.NewGameState(welcomeRes)
    fmt.Printf("Queue %s created. \n", psQueue.Name)

    for {
        if ok := handleLoop(gameState); !ok {
            break
        }
    }
    fmt.Println("Shutting down...")
}

func handleLoop(gs *gamelogic.GameState) bool {
    words := gamelogic.GetInput()
    // Returns false when exit command is given
    for _, w := range words {
    switch w {
        case "quit":
            gamelogic.PrintQuit()
            return false
        case "spawn":
            if err := gs.CommandSpawn(words); err != nil {
                fmt.Println("Bad spawn command")
            }
            return true
        case "move":
            _,err := gs.CommandMove(words)
            if err != nil {
                fmt.Println("Command failed")
            } else {
                fmt.Println("Command successful")
            }
            return true
        case "status":
            gs.CommandStatus()
        case "help":
            gamelogic.PrintClientHelp()
        case "spam":
            fmt.Println("Spamming is not allowed yet!")
        default:
            fmt.Println("Unrecognised command")
        }
    }
    return true
}
