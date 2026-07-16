package airouter

import "net/http"

// HTTPDoer is the package-local protocol boundary used by OpenAI-compatible
// adapters. Implementations must honor request contexts.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type ServiceOptions struct {
	HTTPDoer HTTPDoer
}

var sharedHTTPDoer HTTPDoer = &http.Client{Transport: http.DefaultTransport}

func normalizeHTTPDoer(doer HTTPDoer) HTTPDoer {
	if doer != nil {
		return doer
	}
	return sharedHTTPDoer
}
