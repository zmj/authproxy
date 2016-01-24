package main

import (
	"encoding/json"
	//	"fmt"
	"io"
	"net/url"
	"time"
)

var (
	ExpireLongPoll = 60 * time.Second
	ExpireAuth     = 10 * time.Minute
	Cleanup        = 1 * time.Minute
)

type AuthId string
type AuthContent url.Values

type PollRequest struct {
	Id       AuthId
	Response chan<- *PollResponse
	Received time.Time
}

type PollResponse struct {
	Found   bool
	Content AuthContent
}

type NewAuthRequest struct {
	Id AuthId
}

type AuthSuccess struct {
	Id      AuthId
	Content AuthContent
}

type Cache struct {
	PollRequests    chan *PollRequest
	NewAuthRequests chan *NewAuthRequest
	AuthResponses   chan *AuthSuccess
}

type Auth struct {
	Id       AuthId
	Started  time.Time
	Requests []*PollRequest
	Finished time.Time
	Content  AuthContent
}

func (a *Auth) IsFinished() bool {
	if a == nil {
		return false
	}
	return !a.Finished.IsZero()
}

func (a *Auth) Finish(resp *AuthSuccess) {
	a.Finished = time.Now()
	a.Content = resp.Content
}

func (a *Auth) ExpiresAt() time.Time {
	if a.IsFinished() {
		return a.Finished.Add(ExpireAuth)
	}
	return a.Started.Add(ExpireAuth)
}

func (a *Auth) SuccessResponse() *PollResponse {
	return &PollResponse{Found: true, Content: a.Content}
}

func (a *Auth) SendTimeouts() {
	for len(a.Requests) > 0 && a.Requests[0].Received.Add(ExpireLongPoll).Before(time.Now()) {
		a.Requests[0].Response <- &PollResponse{Found: false}
		a.Requests = a.Requests[1:]
	}
}

func NewCache() *Cache {
	cache := &Cache{
		PollRequests:    make(chan *PollRequest),
		NewAuthRequests: make(chan *NewAuthRequest),
		AuthResponses:   make(chan *AuthSuccess),
	}
	go cache.Cache()
	return cache
}

func (c *Cache) Cache() {
	cleanup := time.Tick(Cleanup)
	auths := make(map[AuthId]*Auth)
	for {
		select {
		case req := <-c.PollRequests:
			req.Received = time.Now()
			auth, exists := auths[req.Id]
			if auth.IsFinished() {
				req.Response <- auth.SuccessResponse()
			} else if exists {
				auth.Requests = append(auth.Requests, req)
			} else {
				auths[req.Id] = &Auth{Id: req.Id, Started: time.Now()}
			}
		case req := <-c.NewAuthRequests:
			if _, exists := auths[req.Id]; !exists {
				auths[req.Id] = &Auth{Id: req.Id, Started: time.Now()}
			}
		case authResp := <-c.AuthResponses:
			auth, exists := auths[authResp.Id]
			if !exists {
				continue
			}
			auth.Finish(authResp)
			for _, req := range auth.Requests {
				req.Response <- auth.SuccessResponse()
			}
			auth.Requests = nil
		case <-cleanup:
			for id, auth := range auths {
				auth.SendTimeouts()
				if auth.ExpiresAt().Before(time.Now()) {
					delete(auths, id)
				}
			}
		}
	}
}

func ParseId(s string) (AuthId, error) {
	return AuthId(s), nil
}

func (c *AuthContent) WriteTo(wr io.Writer) (int64, error) {
	bytes, err := json.Marshal(c)
	if err != nil {
		return 0, err
	}
	written, err := wr.Write(bytes)
	return int64(written), err
}
