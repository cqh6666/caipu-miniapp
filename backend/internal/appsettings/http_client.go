package appsettings

import "net/http"

// HTTPDoer is the protocol boundary for runtime configuration probes.
// Implementations must honor request contexts.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type RuntimeProviderOptions struct {
	HTTPDoer HTTPDoer
}

var sharedHTTPDoer HTTPDoer = &http.Client{Transport: http.DefaultTransport}

func normalizeHTTPDoer(doer HTTPDoer) HTTPDoer {
	if doer != nil {
		return doer
	}
	return sharedHTTPDoer
}
