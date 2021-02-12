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

type Receiver interface {
	Close()
	ServeRequestsLoop()
}

type receiver struct {
	// connection to RabbitMQ
	conn *amqp.Connection
	// queue to consume from
	queueName string
	// webserver endpoint to send requests to
	webEndpoint string

	// receiver uses a goroutine to handle resplying back to rabbitMQ
	// replierQueue is the internal work queue for it
	replierQueue chan *output

	// closure of this channel signals for all go-routines to terminate
	close chan struct{}
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

// ServeRequestsLoop, for better performance/concurrency uses two go routines.
// - one for receiving the requests and issuing them against the webEndpoint
// - another for reading the server output and forwarding that out to rabbitMQ
func (recv *receiver) ServeRequestsLoop() {
	channel, msgQueue, err := ConsumeFromRabbitMQ(recv.conn, recv.queueName)
	FailOnError(err, "Failed to register a consumer")
	defer channel.Close()

	log.Printf(" [*] Waiting for messages on (%v), forwarding to: %v. To exit press CTRL+C",
		recv.queueName, recv.webEndpoint)
	go func() {
		for {
			select {
			case o := <-recv.replierQueue:
				reply(channel, o)
			case <-recv.close:
				log.Printf(" [*] responder goroutine terminating")
				return
			}
		}
	}()

	// queue consumer loop
	client := http.Client{}
LOOP:
	for {
		select {
		case <-recv.close:
			log.Printf(" [*] receiver gorouting terminating.")
			close(recv.replierQueue)
			break LOOP
		case d := <-msgQueue:
			log.Printf("Received a message: %v, body len: %v", d.MessageId, len(d.Body))
			reqPath := d.AppId
			// reqMethod := d.Type // FIXME: support GET/PUT/...
			r, err := client.Post(recv.webEndpoint+reqPath, d.ContentType, bytes.NewReader(d.Body))
			recv.replierQueue <- &output{
				Response:    r,
				Err:         err,
				DeliveryTag: d.DeliveryTag,
				ReplyTo:     d.ReplyTo,
				ID:          d.MessageId,
			}

		}
	}
	log.Printf(" [DONE] finished consumption\n")
}

// Close releases all receiver resources
func (recv *receiver) Close() {
	recv.conn.Close()
	close(recv.close)
}

func NewReceiver(rabbitmqURL, queueName, webEndpoint string) (*receiver, error) {
	conn, err := amqp.Dial(rabbitmqURL)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to connect to RabbitMQ: %v", rabbitmqURL)
	}

	log.Printf("NewReceiver(rabbitMQ: %v, queueName: %v)\n", rabbitmqURL, queueName)
	return &receiver{
		queueName:    queueName,
		webEndpoint:  webEndpoint,
		replierQueue: make(chan *output),
		close:        make(chan struct{}),
		conn:         conn,
	}, nil
}
