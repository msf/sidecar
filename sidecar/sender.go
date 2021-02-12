package sidecar

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"

	"github.com/msf/sidecar/mailbox"
)

// sender is an http proxy sidecar, it pretends to be an http server and
// proxies all http requests to a RabbitMQ persistent queue
// It is complemented by a receiver sidecar that does the reverse.
type sender struct {
	mailbox     *mailbox.Mailbox
	rabbitmq    *amqp.Channel
	conn        *amqp.Connection
	knownQueues map[string]struct{}
}

// SenderRequest mimmicks the data needed for doing an HTTP request
type SenderRequest struct {
	// fields mimmick their HTTP equivalents
	Host        string
	Method      string
	Path        string
	Body        []byte
	ContentType string

	// ID is required for Sender to identify the response that is sent in a different msg queue
	ID string
	// Timeout for request/response to complete
	Timeout time.Duration
}

func (req *SenderRequest) HasError() error {
	if req.ID == "" {
		return errors.New("must have ID")
	}
	if req.Host == "" {
		return errors.New("must have Host")
	}
	if req.Path == "" {
		return errors.New("must have Path")
	}
	return nil
}

type SenderResponse struct {
	StatusCode int
	Body       []byte
}

// Sender is an "sidecar" that almost implements
// a http.Client capable of doing POST/GET/PUT requests
// using rabbitMQ queues and sidecar processes
type Sender interface {
	// Do a request + response using RabbitMQ queues
	Do(req SenderRequest) (*SenderResponse, error)
	// Release all resources
	Close()
}

func (s *sender) Do(req SenderRequest) (*SenderResponse, error) {
	if err := req.HasError(); err != nil {
		return nil, errors.Wrapf(err, "Sender.Do(), invalid req: %+v", req)
	}
	ch, err := s.mailbox.Register(req.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "Sender.Do() requires unique id")
	}
	defer s.mailbox.Deregister(req.ID)

	// if deployment was never active, queue might not exist, we must declare it
	s.declareQueueIfNeeded(req.Host)

	publish := amqp.Publishing{
		Body:        req.Body,
		ContentType: req.ContentType,
		MessageId:   req.ID,
		ReplyTo:     RabbitMQDirectyReplyQueueName,
		Type:        req.Method,
		AppId:       req.Path,
	}
	err = s.rabbitmq.Publish(
		"",       // exchange
		req.Host, // routing key
		false,    // mandatory
		false,    // immediate
		publish,
	)
	if err != nil {
		bodyLen := len(publish.Body)
		publish.Body = []byte{} // redact the body
		msg := errors.Wrapf(err, "ERROR on rabbitmq.ch.Publish() publish: %+v, bodyLen: %v\n", publish, bodyLen)
		log.Print(msg)
		return nil, msg
	}
	log.Printf(" [w] Do() [%v]/%v req %v, body: %v bytes, waiting for reply\n", req.Host, req.Path, req.ID, len(req.Body))

	resp := waitForResponse(ch, req.Timeout)
	if resp == nil {
		return nil, fmt.Errorf("timedout after %v waiting for %v response", req.Timeout, req.ID)
	}
	log.Printf(" [y] Do() req %v, reply: %v bytes\n", req.ID, len(resp.Body))
	return &SenderResponse{
		StatusCode: resp.StatusCode,
		Body:       resp.Body,
	}, resp.Err
}

func waitForResponse(ch <-chan mailbox.Response, timeout time.Duration) *mailbox.Response {
	var resp mailbox.Response
	if timeout <= 0 {
		resp = <-ch
		return &resp
	}

	select {
	case resp = <-ch:
		return &resp
	case <-time.After(timeout):
		return nil
	}
}

func (s *sender) ConsumeReplyQueueLoop() {
	replies, err := ConsumeFromReplyQueue(s.rabbitmq)
	FailOnError(err, "Cannot consume from reply Queue")
	for reply := range replies {
		ch, err := s.mailbox.GetChannel(reply.MessageId)
		if err != nil {
			err := errors.Wrapf(err, "ERROR on mailbox.GetChannel for reply %+v, timedout?", reply)
			log.Print(err)
			continue
		}
		log.Printf("consumeLoop() recv reply for %v, %v bytes\n", reply.MessageId, len(reply.Body))

		// sidecar encodes the http response StatusCode in the reply.Type field
		statusCode, err := strconv.Atoi(reply.Type)
		ch <- mailbox.Response{
			StatusCode: statusCode,
			Body:       reply.Body,
			Err:        err,
		}
	}
	log.Print("ConsumeReplyQueueLoop() END")
}

// HandleWeb is an http endpoint that calls "s.Do(r *http.Request)"
func (s *sender) HandleWeb(w http.ResponseWriter, r *http.Request) {
	// we'll need a unique ID to find the response to our message
	uid := RandStr()
	defer func(startTime time.Time) {
		log.Printf(" [x] ENQUEUE req %v END %v microsecs", uid, time.Since(startTime).Microseconds())
	}(time.Now())
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		msg := fmt.Sprintf(" ERROR on ReadAll(r.Body) err: %v, r: %+v", err, r)
		log.Print(msg)
		fmt.Fprint(w, msg)
		return
	}
	resp, err := s.Do(SenderRequest{
		Path:        r.URL.EscapedPath(),
		Method:      r.Method,
		Body:        body,
		ContentType: r.Header.Get("Content-Type"),
		ID:          uid,
	})
	if err != nil {
		w.WriteHeader(500)
		log.Print(err)
		fmt.Fprint(w, err)
		return
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(resp.Body)
}

func (s *sender) Close() {
	s.conn.Close()
	s.rabbitmq.Close()
}

func (s *sender) declareQueueIfNeeded(queueName string) error {

	if _, ok := s.knownQueues[queueName]; ok {
		return nil
	}
	s.knownQueues[queueName] = struct{}{}

	_, err := s.rabbitmq.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return errors.Wrap(err, "Failed to declare sender queue: "+queueName)
	}
	return nil
}

func NewSender(rabbitmqURL string) (*sender, error) {
	log.Printf("NewSender(rabbitMQ: %v\n", rabbitmqURL)
	conn, err := amqp.Dial(rabbitmqURL)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to connect to RabbitMQ: "+rabbitmqURL)
	}

	rabbitmq, err := conn.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to open rabbitmq channel")
	}

	s := &sender{
		conn:        conn,
		rabbitmq:    rabbitmq,
		mailbox:     mailbox.New(),
		knownQueues: make(map[string]struct{}),
	}
	// FIXME: waitGroup to terminate this goroutine on s.Close()
	go s.ConsumeReplyQueueLoop()
	return s, nil
}
