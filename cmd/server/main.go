package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)
func main() {
	connString := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connString)
	if err != nil {
		log.Fatal("Error creating connexion to rabbitmq server")
	}
	defer conn.Close()
	fmt.Println("Starting Peril server...")

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("Error creating the connection channel")
	}
	defer ch.Close()


	gamelogic.PrintServerHelp()
	loop:
	for {
		var msg []byte
		input := gamelogic.GetInput()
		if len(input) == 0{
			continue
		}
		switch input[0]{
		case "pause":
			ps := routing.PlayingState{
				IsPaused: true,
			} 
			msg, err = json.Marshal(ps)
			if err != nil {
				log.Fatal(err.Error())
			}
			fmt.Println("Pausing the game")
		case "resume":
			rs := routing.PlayingState{
				IsPaused: false,
			}
			msg, err = json.Marshal(rs)
			if err != nil {
				log.Fatal(err.Error())
			}
			fmt.Println("Resuming the game")
		case "quit":
			fmt.Println("Quitting the program")
			break loop
		default:
			fmt.Println("Unknown Command")
		}
		err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, msg)
		if err != nil {
			log.Fatal("Error sending the Pausing message")
		}
	}

}
