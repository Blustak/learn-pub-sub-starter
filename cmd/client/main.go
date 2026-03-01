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

	userName, err := gamelogic.ClientWelcome()
	if err != nil {
		panic(err)
	}
	//Bind a moves queue
	gameState := gamelogic.NewGameState(userName)
	if err = pubsub.SubscribeJSON[gamelogic.ArmyMove](
		conn,
		"peril_topic",
		"army_moves."+userName,
		"army_moves.*",
		pubsub.Transient,
		handleMove(gameState, rabbitmqChannel),
	); err != nil {
		panic(err)
	}

	if err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		"pause."+userName,
		routing.PauseKey,
		pubsub.Transient,
		handlePause(gameState),
	); err != nil {
        panic(err)
    }

    if err = pubsub.SubscribeJSON(
        conn,
        "peril_topic",
        "war",
        routing.WarRecognitionsPrefix + ".*",
        pubsub.Durable,
        handleAllWarMessages(gameState, rabbitmqChannel),
    ); err != nil {
        panic(err)
    }

	for {
		if ok := handleLoop(gameState, rabbitmqChannel, userName); !ok {
			break
		}
	}
	fmt.Println("Shutting down...")
}

func handleLoop(gs *gamelogic.GameState, ch *amqp.Channel, name string) bool {
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
			mv, err := gs.CommandMove(words)
			if err != nil {
				fmt.Println("Command failed")
				return true
			}
			if err := pubsub.PublishJSON(ch, "peril_topic", "army_moves."+name, mv); err != nil {
				fmt.Printf("error publishing move:%s\n", err)
				return true
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

func handlePause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.SimpleAckType {
	return func(r routing.PlayingState) pubsub.SimpleAckType {
		defer fmt.Print("> ")
		gs.HandlePause(r)
		return pubsub.SimpleAckType(pubsub.Ack)
	}
}

func handleMove(gs *gamelogic.GameState, ch *amqp.Channel) func(mv gamelogic.ArmyMove) pubsub.SimpleAckType {
	return func(mv gamelogic.ArmyMove) pubsub.SimpleAckType {
		defer fmt.Print("> ")
		mvOutcome := gs.HandleMove(mv)
		switch mvOutcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.SimpleAckType(pubsub.Ack)
		case gamelogic.MoveOutcomeMakeWar:
            if err := pubsub.PublishJSON(
				ch,
				"peril_topic",
				routing.WarRecognitionsPrefix+"."+gs.GetUsername(),
				gamelogic.RecognitionOfWar{
					Attacker: mv.Player,
					Defender: gs.GetPlayerSnap(),
				},
			); err != nil {
                return pubsub.SimpleAckType(pubsub.NackRequeue)
            }
			return pubsub.SimpleAckType(pubsub.Ack)
		default:
			return pubsub.SimpleAckType(pubsub.NackDiscard)
		}
	}
}

func handleAllWarMessages(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.SimpleAckType {
	return func(dw gamelogic.RecognitionOfWar) pubsub.SimpleAckType {
		defer fmt.Print("> ")
		outcome, winner, loser := gs.HandleWar(dw)
        logMessage := routing.GameLog{
            Username: gs.GetUsername(),
        }
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.SimpleAckType(pubsub.NackRequeue)
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.SimpleAckType(pubsub.NackDiscard)
        case gamelogic.WarOutcomeOpponentWon, gamelogic.WarOutcomeYouWon:
            logMessage.Message = fmt.Sprintf("%s won against %s", winner, loser)
            publishGameLog(ch,logMessage)
			return pubsub.SimpleAckType(pubsub.Ack)
        case gamelogic.WarOutcomeDraw:
            logMessage.Message = fmt.Sprintf("A war between %s and %s resulted in a draw")
            publishGameLog(ch,logMessage)
			return pubsub.SimpleAckType(pubsub.Ack)
        default:
            fmt.Println("Failed to process recognition of war")
            return pubsub.SimpleAckType(pubsub.NackDiscard)
		}
	}
}

func publishGameLog(ch *amqp.Channel, val routing.GameLog) pubsub.SimpleAckType {
    if err := pubsub.PublishGob(
        ch,
        "peril_topic",
        routing.GameLogSlug + "." + val.Username,
        val,
    ); err != nil {
        return pubsub.SimpleAckType(pubsub.NackRequeue)
    }
    return pubsub.SimpleAckType(pubsub.Ack)
}
