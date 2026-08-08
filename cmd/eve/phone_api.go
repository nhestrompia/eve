package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func (server runtimeServer) handlePhoneStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	status, err := server.phone.status(r.Context())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (server runtimeServer) handlePhoneSubscriptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var input registerPhoneSubscriptionInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeAPIError(w, http.StatusBadRequest, fmt.Errorf("decode phone subscription: %w", err))
		return
	}
	device, err := server.phone.register(r.Context(), input, server.phoneRequestUser(r))
	if err != nil {
		writeAPIError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusCreated, device)
}

func (server runtimeServer) handlePhoneSubscriptionRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeMethodNotAllowed(w)
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/phone/subscriptions/"), "/")
	if id == "" || strings.Contains(id, "/") {
		writeAPIError(w, http.StatusNotFound, errors.New("phone subscription route not found"))
		return
	}
	if err := server.phone.removeSubscription(id, server.phoneRequestUser(r)); err != nil && !errors.Is(err, os.ErrNotExist) {
		writeAPIError(w, http.StatusUnprocessableEntity, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (server runtimeServer) handlePhoneTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var input struct {
		SubscriptionID string `json:"subscriptionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(input.SubscriptionID) == "" {
		writeAPIError(w, http.StatusBadRequest, errors.New("subscriptionId is required"))
		return
	}
	if err := server.phone.sendTest(r.Context(), strings.TrimSpace(input.SubscriptionID)); err != nil {
		writeAPIError(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"delivered": true})
}

func (server runtimeServer) phoneRequestUser(r *http.Request) string {
	if login := strings.TrimSpace(r.Header.Get(phoneTailscaleLoginHeader)); login != "" {
		return login
	}
	config, err := server.phone.store.load()
	if err != nil {
		return ""
	}
	return config.AllowedTailscaleLogin
}
