package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/msf/sidecar/sidecar"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	fs := flag.NewFlagSet("sender", flag.ExitOnError)
	var (
		rabbitmqURL = fs.String("rabbitmq_url", "amqp://guest:guest@localhost:5672/", "RabbitMQ URL")
	)
	_ = fs.Parse(os.Args[1:])
	sender, err := sidecar.NewSender(*rabbitmqURL)
	sidecar.FailOnError(err, "Failed to create sender")

	log.Printf(" [*] Waiting for requests, amqp: [%v], To exit press CTRL+C",
		*rabbitmqURL,
	)
	http.HandleFunc("/", sender.HandleWeb)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
