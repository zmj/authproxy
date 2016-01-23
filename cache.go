package main

import (
	"encoding/json"
	//	"fmt"
	"io"
	"net/url"
	"strconv"
	"time"
)

var (
	ExpireLongPoll = 60 * time.Second
	ExpireAuth     = 10 * time.Minute
	Cleanup        = 1 * time.Minute
)

type AuthId int
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
	Response chan<- AuthId
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

func NewAuth() *Auth {
	return &Auth{
		Id:      NewId(),
		Started: time.Now(),
	}
}

func (a *Auth) AddRequest(poll *PollRequest) *Auth {
	if a == nil {
		a = NewAuth()
	}
	a.Requests = append(a.Requests, poll)
	return a
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
			auth := auths[req.Id]
			if auth.IsFinished() {
				req.Response <- auth.SuccessResponse()
			} else {
				auths[req.Id] = auth.AddRequest(req)
			}
		case authReq := <-c.NewAuthRequests:
			auth := NewAuth()
			auths[auth.Id] = auth
			authReq.Response <- auth.Id
		case authResp := <-c.AuthResponses:
			auth := auths[authResp.Id]
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

var lastAuthId = 0 // randomize this

func NewId() AuthId {
	lastAuthId += 1
	return AuthId(lastAuthId)
}

func ParseId(s string) (AuthId, error) {
	i, err := strconv.Atoi(s)
	return AuthId(i), err
}

func (c *AuthContent) WriteTo(wr io.Writer) (int64, error) {
	bytes, err := json.Marshal(c)
	if err != nil {
		return 0, err
	}
	written, err := wr.Write(bytes)
	return int64(written), err
}
