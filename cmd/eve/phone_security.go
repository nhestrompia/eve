package main

import (
	"errors"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const phoneTailscaleLoginHeader = "Tailscale-User-Login"

func (server runtimeServer) phoneAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if server.phone == nil || server.phone.store == nil {
			next.ServeHTTP(w, r)
			return
		}
		config, err := server.phone.store.load()
		if errors.Is(err, os.ErrNotExist) || !config.Enabled {
			next.ServeHTTP(w, r)
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusServiceUnavailable, err)
			return
		}
		if phoneLocalHost(r.Host) {
			next.ServeHTTP(w, r)
			return
		}
		expectedHost := strings.ToLower(net.JoinHostPort(strings.TrimSuffix(config.ServeHost, "."), strconv.Itoa(config.ServePort)))
		if strings.ToLower(strings.TrimSuffix(strings.TrimSpace(r.Host), ".")) != expectedHost {
			writeAPIError(w, http.StatusMisdirectedRequest, errors.New("request host is not configured for EVE phone access"))
			return
		}
		login := strings.TrimSpace(r.Header.Get(phoneTailscaleLoginHeader))
		if login == "" || !strings.EqualFold(login, strings.TrimSpace(config.AllowedTailscaleLogin)) {
			writeAPIError(w, http.StatusForbidden, errors.New("Tailscale identity is not authorized for EVE phone access"))
			return
		}
		if !phoneForwardedHTTPS(r.Header) {
			writeAPIError(w, http.StatusForbidden, errors.New("EVE phone access requires Tailscale HTTPS"))
			return
		}
		if phoneMutationMethod(r.Method) {
			expectedOrigin := config.origin()
			if strings.TrimSuffix(strings.TrimSpace(r.Header.Get("Origin")), "/") != expectedOrigin {
				writeAPIError(w, http.StatusForbidden, errors.New("request origin is not authorized for EVE phone access"))
				return
			}
			fetchSite := strings.TrimSpace(r.Header.Get("Sec-Fetch-Site"))
			if fetchSite != "" && !strings.EqualFold(fetchSite, "same-origin") {
				writeAPIError(w, http.StatusForbidden, errors.New("cross-site mutations are not allowed"))
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		}
		next.ServeHTTP(w, r)
	})
}

func phoneForwardedHTTPS(header http.Header) bool {
	if strings.EqualFold(strings.TrimSpace(header.Get("X-Forwarded-Proto")), "https") {
		return true
	}
	for _, part := range strings.Split(header.Get("Forwarded"), ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && strings.EqualFold(name, "proto") && strings.EqualFold(strings.Trim(value, "\""), "https") {
			return true
		}
	}
	return false
}

func phoneLocalHost(hostport string) bool {
	host := strings.TrimSpace(hostport)
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	} else {
		host = strings.Trim(host, "[]")
	}
	ip := net.ParseIP(host)
	return strings.EqualFold(host, "localhost") || ip != nil && ip.IsLoopback()
}

func phoneMutationMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
