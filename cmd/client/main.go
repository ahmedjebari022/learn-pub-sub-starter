package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")
	const connString = "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connString)	
	if err != nil {
		log.Fatal("Error occured when connecting to the RabbitMq server")
	}
	defer conn.Close()

	userName, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatal("Error when fetching retrieving user Data")
	}

	_, _, err = pubsub.DeclareAndBind(conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+userName,
		routing.PauseKey,
		pubsub.Transient,

	)
	if err != nil {
		log.Fatalf("Error when creating and binding the queu %s",err.Error())
	}

	c := make(chan os.Signal,1)
	signal.Notify(c, os.Interrupt)

	<-c
	fmt.Println("Application interrupted")
	
}
