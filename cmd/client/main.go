package main

import (
	"fmt"
	"log"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

type unitType int
const (
	infantry = iota
	cavalry 
	artillery
)


type locations int
const (
	americas = iota
	europe
	africa
	asia
	antartica
	australia
)

func (l locations) String() string{
	val := []string{
		"americas",
		"europe",
		"africa",
		"asia",
		"antartica", 
		"australia",
	}
	return val[l]
}

func (u unitType) String() string{
	val := []string{
		"infantry",
		"cavalry",
		"artillery",
	}
	return val[u]
}

func checkValidType(ut string) bool {
	vals := []unitType{
		infantry,
		cavalry,
		artillery,
	}
	for _, val := range vals{
		if ut == val.String(){
			return true
		}
	}
	return false
}

func checkValidLocations(l string) bool {
	vals := []locations{
		americas,
		europe,
		africa,
		asia,
		antartica,
		australia,
	}
	for _,val := range vals{
		if l == val.String(){
			return true
		}
	}
	return false
}

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
	
	//declaring moving queu
	ch, _, err := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		"army_moves."+userName,
		"army_moves.*",
		pubsub.Transient,

	)
	if err != nil {
		log.Fatalf("Error when creating and binding the queu %s",err.Error())
	}

	gs := gamelogic.NewGameState(
		userName,	
	)
	
	//binding to moving queu
	pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, "army_moves."+userName, "army_moves.*", pubsub.Transient, handlerMove(gs, ch))
	pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, "pause."+gs.GetUsername(), routing.PauseKey, pubsub.Transient, handlerPause(gs))
	//binding to war uqueu
	pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, "war", routing.WarRecognitionsPrefix+"."+gs.GetUsername(), pubsub.Durable, handlerWar(gs))
	gamelogic.PrintClientHelp()
	loop:
	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			fmt.Println("missing command")
			continue
		}
		switch input[0]{
		case "spawn":
			if !checkValidLocations(input[1]) && !checkValidType(input[2]){
				continue
			}
			err := gs.CommandSpawn(input)
			if err != nil {
				fmt.Printf("error while spawning :%s\n",err.Error())
				continue
			}
		case "move": 
			
			mv, err := gs.CommandMove(input)
			if err != nil {
				fmt.Printf("error whilke moving the unit: %s\n", err.Error())
				continue
			}
			err = pubsub.PublishJSON(ch, routing.ExchangePerilTopic, "army_moves."+gs.GetUsername(), mv)
			if err != nil {
				fmt.Printf("error while publishing to queue: %s\n",err.Error())
			}
		case "status":
			gs.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			break loop
		default: 
			fmt.Println("Invalid command!")
		}
}
	
}

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.Acktype{
	return func(ps routing.PlayingState) pubsub.Acktype {
		defer fmt.Println("> ")
		fmt.Printf("%v\n",ps)
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}


func handlerMove(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.ArmyMove) pubsub.Acktype{
	return func(mv gamelogic.ArmyMove) pubsub.Acktype{
		defer fmt.Println("> ")
		fmt.Println("e")
		mo := gs.HandleMove(mv)
		switch mo{
		case 0:
			return pubsub.NackDiscard
		case 1:
			return pubsub.Ack
		case 2:
			rw := gamelogic.RecognitionOfWar{
				Attacker: mv.Player,
				Defender: gs.Player,
			}
			err := pubsub.PublishJSON(ch, routing.ExchangePerilTopic, routing.WarRecognitionsPrefix +"."+gs.GetUsername(),rw )
			if err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		}
		return pubsub.NackDiscard
	}
}


func handlerWar(gs *gamelogic.GameState) func( gamelogic.RecognitionOfWar) pubsub.Acktype{
	return func(rw gamelogic.RecognitionOfWar) pubsub.Acktype{
		defer fmt.Print("> ")
		o, _, _ := gs.HandleWar(rw)
		switch o {
		case 0: 
			return pubsub.NackRequeue
		case 1:
			return pubsub.NackDiscard
		case 2:
			return pubsub.Ack
		case 3:
			return pubsub.Ack
		case 4:
			return pubsub.Ack
		}
		fmt.Printf("error while handling war")
		return pubsub.NackDiscard
	}
}