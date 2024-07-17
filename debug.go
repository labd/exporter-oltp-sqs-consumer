package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
)

func NewDebugTransport(innerTransport http.RoundTripper) (http.RoundTripper, error) {
	return &LogTransport{
		transport: innerTransport,
	}, nil
}

type LogTransport struct {
	transport http.RoundTripper
}

var DebugTransport = &LogTransport{
	transport: http.DefaultTransport,
}

func (c *LogTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	logRequest(request.Context(), request)
	response, err := c.transport.RoundTrip(request)
	logResponse(request.Context(), response, err)
	return response, err
}

const logRequestTemplate = `DEBUG:
---[ REQUEST ]--------------------------------------------------------
%s
----------------------------------------------------------------------
`

const logResponseTemplate = `DEBUG:
---[ RESPONSE ]-------------------------------------------------------
%s
----------------------------------------------------------------------
`

func logRequest(ctx context.Context, r *http.Request) {
	body, err := httputil.DumpRequestOut(r, true)
	if err != nil {
		return
	}
	log.Printf(logRequestTemplate, body)
}

func logResponse(ctx context.Context, r *http.Response, err error) {
	if err != nil {
		log.Printf(logResponseTemplate, err)
		return
	}
	body, err := httputil.DumpResponse(r, true)
	if err != nil {
		return
	}
	log.Printf(logResponseTemplate, body)
}
