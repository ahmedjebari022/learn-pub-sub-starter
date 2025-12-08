package pubsub

import (
	"context"
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
	
	queue, err := ch.QueueDeclare(queueName, durable, autoDelete, exclusive, false, nil)	
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


func SubscribeJSON[T any](conn *amqp.Connection, exchange, queuName, key string, queueType SimpleQueueType,	handler func(T)) error {
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
				fmt.Printf("msg body:%s\n",string(d.Body))
				handler(data)
				err = d.Ack(false)
				if err != nil {
					fmt.Printf("error while ackising the msg %s\n", err.Error())
				}
			}
		}()
		return nil 
		
		


}