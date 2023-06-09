package web

import (
	"net/http"

	"github.com/peng225/any-exporter/exporter"
)

type MetricsHandler struct {
	ChildHandler http.Handler
}

func (h MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	exporter.Update()
	h.ChildHandler.ServeHTTP(w, r)
}
