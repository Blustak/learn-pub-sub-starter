package main

import (
	"fmt"
	"os"
	"os/signal"

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
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt)
    <-sigChan
    fmt.Println("Shutting down...")
}
