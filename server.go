package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	CompletePrefix = "/authproxy/complete"
	AuthPrefix     = "/authproxy/auth"

	AuthIdKey      = "apid"
	LoginUrlKey    = "loginurl"
	RedirectUrlKey = "redirecturlkey"

	ClosePage = `<html><body>Done! You may close this page.</body></html>`
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
	loginUrl, err := url.ParseRequestURI(req.FormValue(LoginUrlKey))
	if err != nil {
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}
	c := make(chan AuthId)
	s.Cache.NewAuthRequests <- &NewAuthRequest{c}
	id := <-c
	redirectParamName := req.FormValue(RedirectUrlKey)
	redirectUrl := fmt.Sprintf("http://%v%v?%v=%v", req.Host, CompletePrefix, AuthIdKey, id)

	cookie := &http.Cookie{Name: AuthIdKey, Value: id.String(), HttpOnly: true}
	http.SetCookie(wr, cookie)
	q := loginUrl.Query()
	q.Set(redirectParamName, redirectUrl)
	loginUrl.RawQuery = q.Encode()
	wr.Write([]byte(url.QueryEscape(loginUrl.String())))
}

func (s *Server) Poll(wr http.ResponseWriter, req *http.Request) {
	cookie, err := req.Cookie(AuthIdKey)
	if err != nil {
		http.Error(wr, err.Error(), http.StatusBadRequest)
		return
	}
	id, err := ParseId(cookie.Value)
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

func (s *Server) Auth(wr http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "POST", "PUT":
		s.NewAuth(wr, req)
	case "GET", "":
		s.Poll(wr, req)
	default:
		http.Error(wr, "huh", http.StatusBadRequest)
	}
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
	mux.HandleFunc(AuthPrefix, s.Auth)
	mux.HandleFunc(CompletePrefix, s.Complete)
	return mux
}
