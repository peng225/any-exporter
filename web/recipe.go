package web

import (
	"io"
	"log"
	"net/http"

	"github.com/peng225/any-exporter/exporter"
)

func RecipeHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		RecipePostHandler(w, r)
	case http.MethodDelete:
		RecipeDeleteHandler(w, r)
	default:
		log.Printf("invalid method: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func RecipePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		log.Println("request body is nil")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = exporter.Register(body)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Println("recipe post request completed successfully")

	w.WriteHeader(http.StatusOK)
}

func RecipeDeleteHandler(w http.ResponseWriter, r *http.Request) {
	force := r.URL.Query().Get("force") == "true"
	exporter.Clear(force)

	log.Println("recipe delete request completed successfully")

	w.WriteHeader(http.StatusOK)
}
