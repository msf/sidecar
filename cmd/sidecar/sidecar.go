package main

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/namsral/flag"

	"github.com/msf/sidecar/sidecar"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	fs := flag.NewFlagSet("sidecar", flag.ExitOnError)
	var (
		webURL      = fs.String("web_url", "http://web:8080", "maestro url to call")
		rabbitmqURL = fs.String("rabbitmq_url", "amqp://guest:guest@localhost:5672/", "RabbitMQ URL")
		queueName   = fs.String("queue_name", "test-queue", "queue name")
	)
	fs.Usage = usageFor(fs, os.Args[0]+" [flags]")
	_ = fs.Parse(os.Args[1:])

	recv, err := sidecar.NewReceiver(*rabbitmqURL, *queueName, *webURL)
	sidecar.FailOnError(err, "cannot start sidecar.Receiver")

	recv.ServeRequestsLoop()
}

func usageFor(fs *flag.FlagSet, short string) func() {
	return func() {
		fmt.Fprintf(os.Stderr, "USAGE\n")
		fmt.Fprintf(os.Stderr, "  %s\n", short)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "FLAGS\n")
		w := tabwriter.NewWriter(os.Stderr, 0, 2, 2, ' ', 0)
		fs.VisitAll(func(f *flag.Flag) {
			fmt.Fprintf(w, "\t-%s %s\t%s\n", f.Name, f.DefValue, f.Usage)
		})
		w.Flush()
		fmt.Fprintf(os.Stderr, "\n")
	}
}
