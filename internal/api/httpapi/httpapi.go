// SPDX-License-Identifier: AGPL-3.0-only

// Package httpapi exposes the HTTP API as huma operations, with authentication, authorisation and Problem Details errors.
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"github.com/amanuelmr/wickframe/internal/marketdata"
)

// New registers the API's operations on mux and returns the API, whose
// OpenAPI document is also served at /api/v1/openapi.json.
func New(mux *http.ServeMux, candles marketdata.CandleRepository, instruments map[string]int32, log *slog.Logger) huma.API {
	config := huma.DefaultConfig("Wickframe API", "1")
	config.OpenAPIPath = "/api/v1/openapi"
	config.DocsPath = ""
	config.SchemasPath = ""
	// The default hook adds a $schema field to every response body.
	config.CreateHooks = nil
	api := humago.New(mux, config)
	registerCandles(api, candles, instruments, log)
	return api
}

// Problem is an RFC 9457 Problem Details error response.
type Problem struct {
	Type   string              `json:"type" doc:"Stable identifier of the problem; clients branch on it, never on title or detail." enum:"about:blank,urn:wickframe:problem:bad-request,urn:wickframe:problem:not-found,urn:wickframe:problem:validation-error,urn:wickframe:problem:internal-error"`
	Title  string              `json:"title"`
	Status int                 `json:"status"`
	Detail string              `json:"detail,omitempty"`
	Errors []*huma.ErrorDetail `json:"errors,omitempty"`
}

func (p *Problem) Error() string { return p.Detail }

// GetStatus returns the HTTP status code.
func (p *Problem) GetStatus() int { return p.Status }

// ContentType returns the media type of problem responses.
func (p *Problem) ContentType(_ string) string { return "application/problem+json" }

var problemTypes = map[int]string{
	http.StatusBadRequest:          "urn:wickframe:problem:bad-request",
	http.StatusNotFound:            "urn:wickframe:problem:not-found",
	http.StatusUnprocessableEntity: "urn:wickframe:problem:validation-error",
	http.StatusInternalServerError: "urn:wickframe:problem:internal-error",
}

func init() {
	huma.DefaultArrayNullable = false
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		p := &Problem{Type: "about:blank", Title: http.StatusText(status), Status: status, Detail: msg}
		if t, ok := problemTypes[status]; ok {
			p.Type = t
		}
		if status >= http.StatusInternalServerError {
			// Never leak internals; the handler logs the cause.
			p.Detail = ""
		}
		for _, err := range errs {
			if d, ok := err.(huma.ErrorDetailer); ok {
				p.Errors = append(p.Errors, d.ErrorDetail())
			} else if err != nil && status < http.StatusInternalServerError {
				p.Errors = append(p.Errors, &huma.ErrorDetail{Message: err.Error()})
			}
		}
		return p
	}
}
