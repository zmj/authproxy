package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	PollPrefix     = "/authproxy/poll"
	CompletePrefix = "/authproxy/complete"
	NewAuthPrefix  = "/authproxy/auth"

	AuthIdKey      = "apid"
	LoginUrlKey    = "loginurl"
	RedirectUrlKey = "redirecturlkey"

	ClosePage = `<html><script>window.close();</script></html>`
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
	c := make(chan AuthId)
	s.Cache.NewAuthRequests <- &NewAuthRequest{c}
	id := <-c
	loginUrl, err := url.ParseRequestURI(req.URL.Query().Get(LoginUrlKey))
	if err != nil {
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}
	redirectParamName := req.URL.Query().Get(RedirectUrlKey)
	redirectUrl := fmt.Sprintf("%v%v?%v=%v", req.Host, CompletePrefix, AuthIdKey, id)
	q := loginUrl.Query()
	q.Set(redirectParamName, redirectUrl)
	loginUrl.RawQuery = q.Encode()
	http.Redirect(wr, req, loginUrl.String(), http.StatusFound)
}

func (s *Server) Poll(wr http.ResponseWriter, req *http.Request) {
	id, err := ParseId(req.URL.Query().Get(AuthIdKey))
	if err != nil {
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}
	c := make(chan *PollResponse)
	s.Cache.PollRequests <- &PollRequest{Id: id, Response: c}
	resp := <-c
	if !resp.Found {
		http.Error(wr, "nope", http.StatusUnauthorized)
		return
	}
	resp.Content.WriteTo(wr)
}

func (s *Server) Complete(wr http.ResponseWriter, req *http.Request) {
	values := req.URL.Query()
	id, err := ParseId(values.Get(AuthIdKey))
	if err != nil {
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}
	values.Del(AuthIdKey)
	s.Cache.AuthResponses <- &AuthSuccess{
		Id:      id,
		Content: AuthContent(values),
	}
	io.WriteString(wr, ClosePage)
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(NewAuthPrefix, s.NewAuth)
	mux.HandleFunc(CompletePrefix, s.Complete)
	mux.HandleFunc(PollPrefix, s.Poll)
	return mux
}
