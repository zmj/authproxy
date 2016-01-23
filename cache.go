package main

import (
	"time"
)

var (
	ExpireLongPoll = 60 * time.Second
	ExpireResponse = 10 * time.Minute
	Cleanup        = 1 * time.Minute
)

type PollRequest struct {
	Id       string
	Response chan<- *PollResponse
	Received time.Time
}

type PollResponse struct {
	Found   bool
	Content []byte
}

type NewAuthRequest struct {
	Id       string
	Response chan<- *NewAuthResponse
}

type NewAuthResponse struct {
	Ok bool
}

func NewCache() (chan *PollRequest, chan *NewAuthRequest) {
	poll := make(chan *PollRequest)
	newAuth := make(chan *NewAuthRequest)
	go Cache(poll, newAuth)
	return poll, newAuth
}

type Auth struct {
	Content  []byte
	Received time.Time
}

func Cache(pollRequests chan *PollRequest, newAuthRequests chan *NewAuthRequest) {
	cleanup := time.Tick(Cleanup)
	requests := make(map[string][]*PollRequest)
	responses := make(map[string]*Auth)
	for {
		select {
		case pollReq := <-pollRequests:

		case newAuthReq := <-newAuthRequests:

		case <-cleanup:
			now := time.Now()
			for id, reqs := range requests {
				for len(reqs) > 0 && reqs[0].Received.Add(ExpireLongPoll).Before(now) {
					reqs = reqs[1:]
				}
				if len(reqs) == 0 {
					delete(requests, id)
				} else {
					requests[id] = reqs
				}
			}
			for id, auth := range responses {
				if auth.Received.Add(ExpireResponse).Before(now) {
					delete(responses, id)
				}
			}
		}
	}
}
