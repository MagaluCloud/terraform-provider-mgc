package http

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type RequestIDRoundTripper struct {
	next http.RoundTripper
}

func NewRequestIDRoundTripper(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}

	return &RequestIDRoundTripper{
		next: next,
	}
}

func (rt *RequestIDRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	resp, err := rt.next.RoundTrip(req)

	if resp != nil {
		respRequestID := resp.Header.Get("X-Request-Id")
		requestIDText := ""
		if respRequestID != "" {
			requestIDText = fmt.Sprintf("[REQUEST-ID] %s\n", respRequestID)
		}

		respTraceID := resp.Header.Get("X-Mgc-Trace-Id")
		traceIDText := ""
		if respTraceID != "" {
			traceIDText = fmt.Sprintf("[X-MGC-TRACE-ID] %s\n", respTraceID)
		}

		urlText := fmt.Sprintf("[URL] %s\n", resp.Request.URL)

		tflog.Debug(ctx,
			fmt.Sprintf(
				"\n\n\n================ REQUEST RESPONSE INFO ================\n"+
					"%s"+
					"%s"+
					"%s"+
					"=======================================================\n\n\n",
				urlText,
				requestIDText,
				traceIDText,
			),
		)
	}

	return resp, err
}
