package main

import (
	"fmt"
	"net/http"
)

const (
	PollPrefix     = "/authproxy/poll"
	CompletePrefix = "/authproxy/complete"
	NewAuthPrefix  = "/authproxy/auth"
)

func main() {
	s := &Server{Cache: NewCache()}
	(&http.Server{
		Addr:    ":8426",
		Handler: s.Handler(),
	}).ListenAndServe()
	for {
	}
}

type Server struct {
	Cache *Cache
}

func (s *Server) NewAuth(wr http.ResponseWriter, req *http.Request) {
	fmt.Println("auth")
	http.Error(wr, "nyi", http.StatusNotImplemented)
}

func (s *Server) Poll(wr http.ResponseWriter, req *http.Request) {
	fmt.Println("poll")
	http.Error(wr, "nyi", http.StatusNotImplemented)
}

func (s *Server) Complete(wr http.ResponseWriter, req *http.Request) {
	fmt.Println("complete")
	http.Error(wr, "nyi", http.StatusNotImplemented)
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(NewAuthPrefix, s.NewAuth)
	mux.HandleFunc(CompletePrefix, s.Complete)
	mux.HandleFunc(PollPrefix, s.Poll)
	return mux
}
