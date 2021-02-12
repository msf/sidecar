package sidecar

import (
	"log"
	"math/rand"
	"strconv"

	"github.com/streadway/amqp"
)

// see https://www.rabbitmq.com/direct-reply-to.html
const RabbitMQDirectyReplyQueueName = "amq.rabbitmq.reply-to"

// to load balance consumers
// https://www.rabbitmq.com/confirms.html#channel-qos-prefetch
const RabbitMQMessagePrefetchCount = 1

// ConsumeFromReplyQueue implements https://www.rabbitmq.com/direct-reply-to.html
func ConsumeFromReplyQueue(ch *amqp.Channel) (<-chan amqp.Delivery, error) {
	queue, err := ch.Consume(
		RabbitMQDirectyReplyQueueName,
		"sidecar", // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	return queue, err
}

func ConsumeFromRabbitMQ(conn *amqp.Connection, queueName string) (*amqp.Channel, <-chan amqp.Delivery, error) {
	ch, err := conn.Channel()
	FailOnError(err, "Failed to open a channel")

	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	FailOnError(err, "Failed to declare a queue")

	// we set prefetch to share load between consumers
	err = ch.Qos(
		RabbitMQMessagePrefetchCount, // prefetch count
		0,                            // prefetch size
		false,                        // global
	)
	FailOnError(err, "Failed to set channel prefetch count")

	queue, err := ch.Consume(
		q.Name,    // queue
		"sidecar", // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	return ch, queue, err
}

func FailOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func LogOnError(err error, msg string) {
	if err != nil {
		log.Printf("%s: %v\n", msg, err)
	}
}

func RandStr() string {
	return strconv.FormatInt(rand.Int63(), 10)
}
