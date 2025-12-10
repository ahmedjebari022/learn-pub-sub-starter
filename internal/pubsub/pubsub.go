package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
		msg, err := json.Marshal(val)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", msg)
		err = ch.PublishWithContext(context.Background(),exchange, key, false, false, amqp.Publishing{
			ContentType: "application/json",
			Body: 		 msg,
		})
		if err != nil {
			return err
		}	
		return nil

}

type SimpleQueueType int
const (
	Durable SimpleQueueType = iota
	Transient
)

func (qt SimpleQueueType) String() string {
	switch qt {
	case Durable:
		return "durable"
	case Transient:
		return "transient"
	default: 
		return ""
	}
}



func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error){
	
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	exclusive := queueType == Transient
	durable := queueType == Durable
	autoDelete := queueType == Transient
	
	queue, err := ch.QueueDeclare(queueName, durable, autoDelete, exclusive, false, amqp.Table{
		"x-dead-letter-exchange": "peril_dlx",
	})	
	if err != nil {
		ch.Close()
		return nil, amqp.Queue{}, err
	}
	err = ch.QueueBind(queue.Name, key, exchange, false, nil) 
	if err != nil {
		ch.Close()
		return nil, amqp.Queue{}, err
	}
	return ch, queue, nil

}

type Acktype int
const (
	Ack = iota
	NackRequeue
	NackDiscard
)

func (a Acktype) String() string{
	vals := map[Acktype]string{
		0:"Ack",
		1:"NackRequeue",
		2:"NackDiscard",
	}
	return vals[a]
}

func SubscribeJSON[T any](conn *amqp.Connection, exchange, queuName, key string, queueType SimpleQueueType,	handler func(T) Acktype ) error {
		ch, queue, err := DeclareAndBind(conn, exchange, queuName, key, queueType)
		if err != nil {
			return err
		}
		c, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
		if err != nil {
			return err
		}

		go func(){
			for d := range c {
				var data T
				err := json.Unmarshal(d.Body, &data)
				if err != nil {
					fmt.Printf("error while processing the msg: %s\n", err.Error())
				}
				ack := handler(data)
				switch ack.String() {
				case "Ack" :
					err = d.Ack(false)
					fmt.Printf("Ack")
					if err != nil {
						fmt.Printf("error while ackising the msg %s\n", err.Error())
					}
				case "NackRequeue":
					fmt.Printf("NackRequeue")
					err = d.Nack(false, true)
					if err != nil {
						fmt.Printf("error while ackising the smg %s\n", err.Error())
					}
				case "NackDiscard":
					fmt.Printf("NackDiscard")
					err = d.Nack(false, false)
					if err != nil {
						fmt.Printf("error while ackising the msg%s\n", err.Error())
					}
				}
			}
		}()
		return nil 
}


func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {

	
	var msg bytes.Buffer
	encoder := gob.NewEncoder(&msg)
	err := encoder.Encode(val)
	if err != nil {
		fmt.Println("error while encoding")
		return err
	}

	err = ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/gob",
		Body: msg.Bytes(),
	})
	if err != nil {
		return err
	}
	fmt.Printf("sent message to exchange%s\n",exchange)
	return nil 
}



func SubscribeGob[T any](conn *amqp.Connection, exchange, queueName, key string, simpleQSimpleQueueType SimpleQueueType, handler func(T) Acktype, unmarshaller func([]byte) (T, error)) error{
	ch, _, err := DeclareAndBind(conn, exchange, queueName, key, Durable)
	if err != nil {
		return err
	}
	err = ch.Qos(10,0,true)
	if err != nil {
		return err
	}
	deliveries, err := ch.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func(){
		for d := range deliveries {
			msg, err := unmarshaller(d.Body)	
			if err != nil {
				fmt.Printf("error while decoding the msg: %s\n",err.Error())
			}
			ackType := handler(msg)
			switch ackType.String() {
			case "Ack" :
				err = d.Ack(false)
				fmt.Printf("Ack")
				if err != nil {
					fmt.Printf("error while ackising the msg %s\n", err.Error())
				}
			case "NackRequeue":
				fmt.Printf("NackRequeue")
				err = d.Nack(false, true)
				if err != nil {
					fmt.Printf("error while ackising the smg %s\n", err.Error())
				}
			case "NackDiscard":
				fmt.Printf("NackDiscard")
				err = d.Nack(false, false)
				if err != nil {
					fmt.Printf("error while ackising the msg%s\n", err.Error())
				}
			}
		}
	}()
		return nil
}


func GobDecoder[T any](data []byte) (T, error) {
	var t T
	decoder := gob.NewDecoder(bytes.NewBuffer(data))
	err := decoder.Decode(&t)
	if err != nil {
		return t, err
	}
	return t, nil
} 