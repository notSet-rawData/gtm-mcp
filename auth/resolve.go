package auth

import (
	"net/http"
	"net/url"
)

type URLResolver struct {
	configuredURL  string
	configuredHost string
	allowedHosts   map[string]bool
}

func NewURLResolver(baseURL string, allowedHosts []string) *URLResolver {
	parsed, _ := url.Parse(baseURL)
	configuredHost := ""
	if parsed != nil {
		configuredHost = parsed.Host
	}

	allowed := make(map[string]bool, len(allowedHosts)+1)
	if configuredHost != "" {
		allowed[configuredHost] = true
	}
	for _, h := range allowedHosts {
		allowed[h] = true
	}

	return &URLResolver{
		configuredURL:  baseURL,
		configuredHost: configuredHost,
		allowedHosts:   allowed,
	}
}

func (u *URLResolver) Resolve(r *http.Request) string {
	host := r.Host
	if host == "" {
		return u.configuredURL
	}

	if !u.allowedHosts[host] {
		return u.configuredURL
	}

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	return scheme + "://" + host
}
