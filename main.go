package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/nats-io/stan.go"
)

const (
	DEFAULT_PORT int = 8080
)

type Config struct {
	NatsURL       string
	StanClientID  string
	StanClusterID string
	ListenPort    int
}

func getConfig() *Config {
	port := DEFAULT_PORT
	portEnv := os.Getenv("LISTEN_PORT")
	if p, err := strconv.Atoi(portEnv); err != nil {
		if p > 0 {
			port = p
		}
	}

	return &Config{
		StanClientID:  os.Getenv("STAN_CLIENTID"),
		StanClusterID: os.Getenv("STAN_CLUSTERID"),
		NatsURL:       os.Getenv("NATS_URL"),
		ListenPort:    port,
	}
}
func logCloser(c io.Closer) {
	if err := c.Close(); err != nil {
		log.Printf("close error: %s", err)
	}
}

func logger(message string) {
	log.Printf("%s: %s", time.Now().Format(time.RFC3339), message)
}

func process(msg *stan.Msg) error {
	logger(fmt.Sprintf("Received subject: %s", msg.Subject))
	logger(fmt.Sprintf("Received message: %s", msg.String()))
	logger("Very long processing")
	defer logger("You've done a long long time")
	time.Sleep(30 * time.Second)
	return nil
}

func handle(msg *stan.Msg) {
	err := process(msg)
	if err == nil {
		msg.Ack()
	}
}

func main() {
	config := getConfig()
	opts := []stan.Option{}
	if config.NatsURL != "" {
		opts = append(opts, stan.NatsURL(config.NatsURL))
	}

	if config.StanClusterID == "" {
		log.Print("STAN_CLUSTERID must be specified")
		os.Exit(2)
	}
	if config.StanClientID == "" {
		log.Print("STAN_CLIENTID must be specified")
		os.Exit(2)
	}

	conn, err := stan.Connect(config.StanClusterID, config.StanClientID, opts...)
	if err != nil {
		log.Print(err)
		os.Exit(2)
	}
	defer logCloser(conn)

	conn.Subscribe(
		"demo",
		handle,
		stan.DurableName("i-will-remember"),
		stan.MaxInflight(1),
		stan.SetManualAckMode(),
	)

	r := mux.NewRouter()
	r.HandleFunc("/healthz", healtz)
	r.HandleFunc("/ready", ready)
	r.HandleFunc("/metrics", metrics)

	listen := fmt.Sprintf(":%d", config.ListenPort)
	http.ListenAndServe(listen, r)
	os.Exit(0)
}

func healtz(w http.ResponseWriter, req *http.Request) {
	http.Error(w, "I'm not live", 503)
}

func ready(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, "I'm ready")
}

func metrics(res http.ResponseWriter, req *http.Request) {
	fmt.Fprint(res, "Nothing")
}
