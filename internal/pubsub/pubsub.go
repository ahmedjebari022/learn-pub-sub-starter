package pubsub

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
		msg, err := json.Marshal(val)
		if err != nil {
			return err
		}
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