// Package query provides serialisation of AWS query requests, and responses.
package query

//go:generate go run ../../fixtures/protocol/generate.go ../../fixtures/protocol/input/query.json build_test.go

import (
	"net/url"

	"github.com/mendersoftware/artifacts/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/mendersoftware/artifacts/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/request"
	"github.com/mendersoftware/artifacts/Godeps/_workspace/src/github.com/aws/aws-sdk-go/internal/protocol/query/queryutil"
)

// Build builds a request for an AWS Query service.
func Build(r *request.Request) {
	body := url.Values{
		"Action":  {r.Operation.Name},
		"Version": {r.Service.APIVersion},
	}
	if err := queryutil.Parse(body, r.Params, false); err != nil {
		r.Error = awserr.New("SerializationError", "failed encoding Query request", err)
		return
	}

	if r.ExpireTime == 0 {
		r.HTTPRequest.Method = "POST"
		r.HTTPRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
		r.SetBufferBody([]byte(body.Encode()))
	} else { // This is a pre-signed request
		r.HTTPRequest.Method = "GET"
		r.HTTPRequest.URL.RawQuery = body.Encode()
	}
}
