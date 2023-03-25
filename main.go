package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/peng225/promblock/web"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	port := flag.Int("port", 8080, "listen port")

	flag.Parse()

	log.SetFlags(log.Lshortfile)

	if *port < 0 || *port > 65536 {
		log.Fatalf("Invalid port number: %d", *port)
	}

	metricsHandler := web.MetricsHandler{
		ChildHandler: promhttp.Handler(),
	}

	http.Handle("/metrics", metricsHandler)
	http.HandleFunc("/recipe", web.RecipeHandler)
	err := http.ListenAndServe(net.JoinHostPort("", strconv.Itoa(*port)), nil)
	if err != nil {
		log.Println(err)
	}
}
