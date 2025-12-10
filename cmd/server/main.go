package main

import (
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
	_, _, err = pubsub.DeclareAndBind(conn, "peril_topic", "game_logs", "game_logs.*", pubsub.Durable)
	if err != nil {
		log.Fatal(err.Error())
	} 
	pubsub.SubscribeGob(conn, routing.ExchangePerilTopic, "game_logs", "game_logs.*", pubsub.Durable, logHandler(), pubsub.GobDecoder[routing.GameLog])
	gamelogic.PrintServerHelp()
	loop:
	for {
		input := gamelogic.GetInput()
		if len(input) == 0{
			continue
		}
		ps := routing.PlayingState{}
		switch input[0]{
		case "pause":
			ps = routing.PlayingState{
				IsPaused: true,
			} 
			fmt.Printf("ps:%v\n",ps)
			fmt.Println("Pausing the game")
		case "resume":
			ps = routing.PlayingState{
				IsPaused: false,
			}
			fmt.Println("Resuming the game")
		case "quit":
			fmt.Println("Quitting the program")
			break loop
		default:
			fmt.Println("Unknown Command")
		}
		err = pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, ps)
		if err != nil {
			log.Fatal("Error sending the Pausing message")
		}
	}

}


func logHandler() func(routing.GameLog) pubsub.Acktype {
	return func(gl routing.GameLog) pubsub.Acktype{
		defer fmt.Println(">  ")
		err := gamelogic.WriteLog(gl)
		if err != nil {
			return pubsub.NackRequeue
		}	
		return pubsub.Ack
	}
}