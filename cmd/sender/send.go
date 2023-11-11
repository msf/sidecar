package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/msf/sidecar/sidecar"
	"github.com/namsral/flag"
)

type server struct {
	sender sidecar.Sender
	client *http.Client
}

func (s *server) HandleWeb(w http.ResponseWriter, r *http.Request) {
	uid := sidecar.RandStr()
	host, path := parseHostPath(r.URL)
	defer func(startTime time.Time) {
		log.Printf(" [x] WEB %v/%v req %v took: %v microsecs", host, path, uid, time.Since(startTime).Microseconds())
	}(time.Now())
	resp, err := s.client.Post(
		fmt.Sprintf("http://%v.sidecar.svc.cluster.local/%v", host, path),
		"application/json",
		bytes.NewReader(sidecar.MaestroRequest{
			Text:           "please translate",
			SourceLanguage: "en",
			TargetLanguage: "pt",
			UID:            uid,
		}.ToJSON()),
	)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "ERROR err: %v", err)
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "ERROR err: %v", err)
		return
	}
	out := fmt.Sprintf(" WEB - %v - %v\n", uid, string(body))
	log.Print(out)
	fmt.Fprint(w, out)
}

func (s *server) HandleQueue(w http.ResponseWriter, r *http.Request) {
	uid := sidecar.RandStr()
	host, path := parseHostPath(r.URL)
	defer func(startTime time.Time) {
		log.Printf(" [x] ENQUEUE %v/%v req %v took: %v microsecs", host, path, uid, time.Since(startTime).Microseconds())
	}(time.Now())

	resp, err := s.sender.Do(sidecar.SenderRequest{
		Method: "POST",
		Host:   host,
		Path:   path,
		ID:     uid,
		Body: sidecar.MaestroRequest{
			Text:           "please translate",
			SourceLanguage: "en",
			TargetLanguage: "pt",
			UID:            uid,
		}.ToJSON(),
	})
	if err != nil {
		w.WriteHeader(500)
		msg := fmt.Sprintf("ERROR on %v getting response err: %v, r: %+v", uid, err, r)
		log.Print(msg)
		fmt.Fprint(w, msg)
		return
	}
	out := fmt.Sprintf(" ENQUEUE - %v - %v\n", uid, string(resp.Body))
	log.Print(out)
	fmt.Fprint(w, out)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	fs := flag.NewFlagSet("sender", flag.ExitOnError)
	var (
		rabbitmqURL = fs.String("rabbitmq_url", "amqp://guest:guest@localhost:5672/", "RabbitMQ URL")
	)
	_ = fs.Parse(os.Args[1:])

	sender, err := sidecar.NewSender(*rabbitmqURL)
	sidecar.FailOnError(err, "Failed to create Sender")
	defer sender.Close()

	srv := &server{
		sender: sender,
		client: &http.Client{},
	}
	log.Printf(" [*] Waiting for requests ['0.0.0.0:8080']\n\t amqp: [%v]. To exit press CTRL+C", *rabbitmqURL)

	http.HandleFunc("/q/", srv.HandleQueue)
	http.HandleFunc("/w/", srv.HandleWeb)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func parseHostPath(url *url.URL) (string, string) {
	// TODO: this is simplified and wrong for demo reasons
	parts := strings.Split(url.EscapedPath(), "/")
	// "hostname/queue/en-pt/mt_qe" or "hostname/web/en-es/health"
	return parts[2], "/" + parts[len(parts)-1]
}
