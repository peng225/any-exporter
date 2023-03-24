package main

import (
	"flag"
	"log"
	"net"
	"net/http"

	"github.com/peng225/promblock/web"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	port := flag.String("port", "", "listen port")

	flag.Parse()

	metricsHandler := web.MetricsHandler{
		ChildHandler: promhttp.Handler(),
	}

	http.Handle("/metrics", metricsHandler)
	http.HandleFunc("/recipe", web.RecipeHandler)
	err := http.ListenAndServe(net.JoinHostPort("", *port), nil)
	if err != nil {
		log.Println(err)
	}
}
