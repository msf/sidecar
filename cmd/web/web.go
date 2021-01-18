package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type req struct {
	i     int
	outCh chan float64
}
type server struct {
	workQueue chan *req
	client    *http.Client
}

func (s *server) workerLoop() {
	for req := range s.workQueue {
		req.outCh <- burnCPU(req.i)
		close(req.outCh)
	}
}

func burnCPU(count int) float64 {
	rng := rand.New(rand.NewSource(31))
	var out float64
	for i := 0; i < count; i++ {
		out += rng.NormFloat64()
	}
	return out
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I healthy!")
}

func (s *server) everything(w http.ResponseWriter, r *http.Request) {
	out := fmt.Sprintf(" request: %+v\n", r)
	log.Print(out)
	time.Sleep(200 * time.Millisecond)
	log.Print("DONE\n")
	fmt.Fprint(w, out)
}

func (s *server) mtqe(w http.ResponseWriter, r *http.Request) {
	defer func(startTime time.Time) {
		log.Printf("req took: %v microsecs", time.Since(startTime).Microseconds())
	}(time.Now())
	out := fmt.Sprintf(" MT_QE request: %+v\n", r)
	log.Print(out)
	ch := make(chan float64)
	s.workQueue <- &req{
		i:     29990000,
		outCh: ch,
	}
	val := <-ch
	tmp := fmt.Sprintf(" MT_QE: CPUout: %v, call EQ gRPC endp", val)
	log.Print(tmp)
	fmt.Fprintf(w, "{'text':'%v'}", tmp)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	srv := &server{
		workQueue: make(chan *req),
		client:    &http.Client{},
	}
	go srv.workerLoop()

	log.Print(" [*] Waiting for requests. To exit press CTRL+C")
	http.HandleFunc("/health", srv.health)
	http.HandleFunc("/mt_qe", srv.mtqe)
	http.HandleFunc("/", srv.everything)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
