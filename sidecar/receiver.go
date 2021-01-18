package sidecar

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

type output struct {
	Response    *http.Response
	Err         error
	ReplyTo     string
	DeliveryTag uint64
	ID          string
}

type receiver struct {
	// queue to consume from
	queueName string

	// webserver endpoint to send requests to
	webEndpoint string

	// channel to give work to the replier
	outputQueue chan *output

	conn *amqp.Connection
}

// send a reply back based on output
func reply(ch *amqp.Channel, out *output) {
	var body []byte
	var err error
	var responseMsg amqp.Publishing

	responseMsg.MessageId = out.ID
	if out.Err != nil {
		log.Printf("Failed HTTP protocol to Maestro (will deliver error 599): %v, msg ID: %v\n",
			out.Err, out.ID)
		responseMsg.Body = []byte(out.Err.Error())
		responseMsg.Type = strconv.Itoa(599)
	} else {
		body, err = ioutil.ReadAll(out.Response.Body)
		LogOnError(err, "Read Response.Body: "+out.ID)
		out.Response.Body.Close()
		responseMsg.Body = body
		responseMsg.Type = strconv.Itoa(out.Response.StatusCode)
	}
	// We use an ephemeral callback queue instead of an HTTP callback
	// see https://www.rabbitmq.com/direct-reply-to.html
	err = ch.Publish(
		"",
		out.ReplyTo,
		false, // mandatory
		false, // immediate
		responseMsg,
	)
	if err != nil {
		// This failure case (if possible) can lead to deadlock,
		// the client will be stuck waiting for this out message
		// so we won't ack the original message
		// FIXME (msf): should we logfatal and exit? only when client exits will msg be released
		log.Printf("Failed rabbitmq.Publish() to: %v, err: %v, will NOT ACK %v\n",
			out.ReplyTo,
			err,
			out.ID)
	} else {
		log.Printf("Delivered result for message: %v, len: %v, replyTo: %v\n",
			out.ID, len(responseMsg.Body), out.ReplyTo,
		)
		err = ch.Ack(out.DeliveryTag, false)
		LogOnError(err, fmt.Sprintf("ERROR, failed to ack message: %v, deliveryTag: %v\n",
			out.ID, out.DeliveryTag))
	}
}

// ProcessRequestsLoop, for better performance/concurrency uses two go routines.
// - one for receiving the requests and issuing them against the webEndpoint
// - another for reading the server output and forwarding that out to rabbitMQ
func (recv *receiver) ProcessRequestsLoop() {
	channel, msgQueue, err := ConsumeFromRabbitMQ(recv.conn, recv.queueName)
	FailOnError(err, "Failed to register a consumer")
	defer channel.Close()

	log.Printf(" [*] Waiting for messages on (%v), forwarding to: %v. To exit press CTRL+C",
		recv.queueName, recv.webEndpoint)
	go func() {
		for o := range recv.outputQueue {
			reply(channel, o)
		}
	}()

	// queue consumer loop
	client := http.Client{}
	for d := range msgQueue {
		log.Printf("Received a message: %v, body len: %v", d.MessageId, len(d.Body))
		reqPath := d.AppId
		// reqMethod := d.Type // FIXME: support GET/PUT/...
		r, err := client.Post(recv.webEndpoint+reqPath, d.ContentType, bytes.NewReader(d.Body))
		recv.outputQueue <- &output{
			Response:    r,
			Err:         err,
			DeliveryTag: d.DeliveryTag,
			ReplyTo:     d.ReplyTo,
			ID:          d.MessageId,
		}
	}
	log.Printf("finished consumption\n")
}

func (recv *receiver) Close() {
	recv.conn.Close()
}

func NewReceiver(rabbitmqURL, queueName, webEndpoint string) (*receiver, error) {
	conn, err := amqp.Dial(rabbitmqURL)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to connect to RabbitMQ: %v", rabbitmqURL)
	}

	log.Printf("NewReceiver(rabbitMQ: %v, queueName: %v)\n", rabbitmqURL, queueName)
	return &receiver{
		queueName:   queueName,
		webEndpoint: webEndpoint,
		outputQueue: make(chan *output),
		conn:        conn,
	}, nil
}
