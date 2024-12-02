package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()
	handler := func(w http.ResponseWriter, r *http.Request) {

		rand.Seed(time.Now().UnixNano())
		fmt.Fprintf(w, "%d", rand.Intn(100))
	}
	r.HandleFunc("/", handler)
	s := &http.Server{
		Addr:    ":80",
		Handler: r,
	}
	log.Fatal(s.ListenAndServe())
}
