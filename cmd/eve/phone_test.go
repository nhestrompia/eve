package main

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPhoneConfigIsAtomicPrivateAndReusable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "phone.json")
	store := &phoneConfigStore{path: path}
	first, err := store.update(func(config *phoneConfig) error {
		config.Enabled = true
		config.ServeHost = "mac.example.ts.net"
		config.ServePort = 8443
		return ensurePhoneVAPIDKey(config)
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.VAPIDPrivateKey == "" {
		t.Fatal("VAPID private key was not generated")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("phone config permissions = %o, want 600", got)
	}
	second, err := store.update(func(config *phoneConfig) error { return ensurePhoneVAPIDKey(config) })
	if err != nil {
		t.Fatal(err)
	}
	if second.VAPIDPrivateKey != first.VAPIDPrivateKey {
		t.Fatal("VAPID private key changed during idempotent update")
	}
	if second.origin() != "https://mac.example.ts.net:8443" {
		t.Fatalf("origin = %q", second.origin())
	}
}

func TestPhoneConfigRejectsCorruptionAndUnknownSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "phone.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &phoneConfigStore{path: path}
	if _, err := store.load(); err == nil || !strings.Contains(err.Error(), "parse phone configuration") {
		t.Fatalf("corrupt configuration error = %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":99}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.load(); err == nil || !strings.Contains(err.Error(), "schemaVersion") {
		t.Fatalf("unknown schema error = %v", err)
	}
}

func TestValidatePhoneSubscriptionRequiresAppleAndValidKeys(t *testing.T) {
	input := validPhoneSubscriptionInput(t, "https://web.push.apple.com/QWERTY")
	if _, err := validatePhoneSubscriptionInput(input, func(host string) bool { return strings.HasSuffix(host, ".push.apple.com") }); err != nil {
		t.Fatalf("valid Apple subscription: %v", err)
	}
	input.Endpoint = "https://example.com/push"
	if _, err := validatePhoneSubscriptionInput(input, func(host string) bool { return strings.HasSuffix(host, ".push.apple.com") }); err == nil {
		t.Fatal("non-Apple endpoint was accepted")
	}
	input = validPhoneSubscriptionInput(t, "https://web.push.apple.com/QWERTY")
	input.Keys.P256DH = "not-a-key"
	if _, err := validatePhoneSubscriptionInput(input, func(string) bool { return true }); err == nil {
		t.Fatal("invalid P-256 key was accepted")
	}
}

func TestPhoneAccessMiddlewareEnforcesHostIdentityHTTPSAndOrigin(t *testing.T) {
	store := &phoneConfigStore{path: filepath.Join(t.TempDir(), "phone.json")}
	_, err := store.update(func(config *phoneConfig) error {
		config.Enabled = true
		config.ServeHost = "mac.example.ts.net"
		config.ServePort = 8443
		config.AllowedTailscaleLogin = "owner@example.com"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	server := runtimeServer{phone: newPhoneManager(store, nil)}
	handler := server.phoneAccessMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))

	request := httptest.NewRequest(http.MethodPost, "https://mac.example.ts.net:8443/api/phone/test", strings.NewReader("{}"))
	request.Host = "mac.example.ts.net:8443"
	request.Header.Set(phoneTailscaleLoginHeader, "OWNER@example.com")
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("Origin", "https://mac.example.ts.net:8443")
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("authorized request status = %d body = %s", recorder.Code, recorder.Body.String())
	}

	for name, testCase := range map[string]struct {
		mutate func(*http.Request)
		want   int
	}{
		"unknown host":     {func(r *http.Request) { r.Host = "attacker.example" }, http.StatusMisdirectedRequest},
		"missing identity": {func(r *http.Request) { r.Header.Del(phoneTailscaleLoginHeader) }, http.StatusForbidden},
		"plain proxy":      {func(r *http.Request) { r.Header.Del("X-Forwarded-Proto") }, http.StatusForbidden},
		"cross origin":     {func(r *http.Request) { r.Header.Set("Origin", "https://attacker.example") }, http.StatusForbidden},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := request.Clone(context.Background())
			candidate.Header = request.Header.Clone()
			testCase.mutate(candidate)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, candidate)
			if recorder.Code != testCase.want {
				t.Fatalf("status = %d want %d body = %s", recorder.Code, testCase.want, recorder.Body.String())
			}
		})
	}

	local := httptest.NewRequest(http.MethodPost, "http://localhost:4317/api/anything", nil)
	local.Host = "localhost:4317"
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, local)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("local request status = %d", recorder.Code)
	}
}

func TestPhoneAccessAcceptsStandardForwardedHTTPS(t *testing.T) {
	store := &phoneConfigStore{path: filepath.Join(t.TempDir(), "phone.json")}
	_, err := store.update(func(config *phoneConfig) error {
		config.Enabled = true
		config.ServeHost = "mac.example.ts.net"
		config.ServePort = 8443
		config.AllowedTailscaleLogin = "owner@example.com"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := (runtimeServer{phone: newPhoneManager(store, nil)}).phoneAccessMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "https://mac.example.ts.net:8443/api/phone/status", nil)
	request.Host = "mac.example.ts.net:8443"
	request.Header.Set(phoneTailscaleLoginHeader, "owner@example.com")
	request.Header.Set("Forwarded", `for=100.64.0.1;proto=https;host="mac.example.ts.net:8443"`)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("Forwarded HTTPS status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestPhoneAPIRegistersTestsAndRemovesSubscriptionWithoutSecrets(t *testing.T) {
	var receivedHeader http.Header
	pushServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Clone()
		w.WriteHeader(http.StatusCreated)
	}))
	defer pushServer.Close()

	store := &phoneConfigStore{path: filepath.Join(t.TempDir(), "phone.json")}
	_, err := store.update(func(config *phoneConfig) error {
		config.Enabled = true
		config.ServeHost = "mac.example.ts.net"
		config.ServePort = 8443
		config.AllowedTailscaleLogin = "owner@example.com"
		return ensurePhoneVAPIDKey(config)
	})
	if err != nil {
		t.Fatal(err)
	}
	server := newRuntimeServer(repository{}, "127.0.0.1:4317")
	manager := newPhoneManager(store, func(context.Context) ([]*planRequest, error) { return []*planRequest{}, nil })
	manager.client = pushServer.Client()
	manager.endpointAllowed = func(string) bool { return true }
	server.phone = manager
	handler := server.routes()
	input := validPhoneSubscriptionInput(t, pushServer.URL+"/device-token")
	body, _ := json.Marshal(input)
	recorder := servePhoneRequest(t, handler, http.MethodPost, "/api/phone/subscriptions", body)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("register status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "endpoint") || strings.Contains(recorder.Body.String(), "p256dh") || strings.Contains(recorder.Body.String(), "auth") {
		t.Fatalf("registration response leaked subscription secrets: %s", recorder.Body.String())
	}
	var device phoneDeviceStatus
	if err := json.Unmarshal(recorder.Body.Bytes(), &device); err != nil {
		t.Fatal(err)
	}
	statusRecorder := servePhoneRequest(t, handler, http.MethodGet, "/api/phone/status", nil)
	if statusRecorder.Code != http.StatusOK {
		t.Fatalf("status code = %d body = %s", statusRecorder.Code, statusRecorder.Body.String())
	}
	statusBody := statusRecorder.Body.String()
	for _, secret := range []string{input.Endpoint, input.Keys.P256DH, input.Keys.Auth} {
		if strings.Contains(statusBody, secret) {
			t.Fatalf("phone status leaked subscription secret: %s", statusBody)
		}
	}
	if !strings.Contains(statusBody, `"lastSuccessAt":null`) || !strings.Contains(statusBody, `"lastError":null`) {
		t.Fatalf("phone status must represent absent delivery fields as null: %s", statusBody)
	}

	testBody, _ := json.Marshal(map[string]string{"subscriptionId": device.ID})
	recorder = servePhoneRequest(t, handler, http.MethodPost, "/api/phone/test", testBody)
	if recorder.Code != http.StatusOK {
		t.Fatalf("test push status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	for _, header := range []string{"Authorization", "Content-Encoding", "TTL", "Topic", "Urgency"} {
		if receivedHeader.Get(header) == "" {
			t.Fatalf("push request missing %s", header)
		}
	}
	if receivedHeader.Get("Urgency") != "high" || receivedHeader.Get("TTL") != "86400" {
		t.Fatalf("push headers = %#v", receivedHeader)
	}

	recorder = servePhoneRequest(t, handler, http.MethodDelete, "/api/phone/subscriptions/"+device.ID, nil)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	config, _ := store.load()
	if len(config.Subscriptions) != 0 {
		t.Fatalf("subscriptions remain: %#v", config.Subscriptions)
	}
}

func TestPhoneManagerSeedsExistingPlansAndDeduplicatesDeliveries(t *testing.T) {
	store := &phoneConfigStore{path: filepath.Join(t.TempDir(), "phone.json")}
	_, err := store.update(func(config *phoneConfig) error {
		config.Enabled = true
		config.ServeHost = "mac.example.ts.net"
		config.ServePort = 8443
		config.AllowedTailscaleLogin = "owner@example.com"
		return ensurePhoneVAPIDKey(config)
	})
	if err != nil {
		t.Fatal(err)
	}
	plan := &planRequest{PlanRequestID: "planreq_existing_12345678", CurrentRevision: 1, Repository: "eve"}
	manager := newPhoneManager(store, func(context.Context) ([]*planRequest, error) { return []*planRequest{plan}, nil })
	manager.endpointAllowed = func(string) bool { return true }
	input := validPhoneSubscriptionInput(t, "https://push.example/device")
	device, err := manager.register(context.Background(), input, "owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	config, _ := store.load()
	delivery, found := config.delivery(phoneNotificationKey(plan), device.ID)
	if !found || delivery.State != "skipped_existing" {
		t.Fatalf("existing plan delivery = %#v found=%v", delivery, found)
	}
}

func TestPhoneHelpersPreserveGraphemesAndTopicLimits(t *testing.T) {
	goal := strings.Repeat("👩🏽‍💻", 121)
	got := truncatePhoneGoal(goal, 120)
	if !strings.HasSuffix(got, "…") || strings.Count(got, "👩🏽‍💻") != 120 {
		t.Fatalf("truncated goal did not preserve graphemes: %q", got)
	}
	if topic := phoneTopic("planreq:1"); len(topic) != 32 {
		t.Fatalf("topic length = %d", len(topic))
	}
}

type fakePhoneRunner struct {
	mu         sync.Mutex
	configured bool
	calls      []string
}

func (runner *fakePhoneRunner) Run(name string, args []string, _ []string) ([]byte, error) {
	runner.mu.Lock()
	defer runner.mu.Unlock()
	call := filepath.Base(name) + " " + strings.Join(args, " ")
	runner.calls = append(runner.calls, call)
	switch {
	case strings.Contains(call, "tailscale version"):
		return []byte("1.80.2\n"), nil
	case strings.Contains(call, "tailscale status --json"):
		return []byte(`{"BackendState":"Running","Self":{"DNSName":"mac.example.ts.net.","UserID":7},"User":{"7":{"LoginName":"owner@example.com"}}}`), nil
	case strings.Contains(call, "tailscale serve status --json"):
		if runner.configured {
			return []byte(`{"Web":{"mac.example.ts.net:8443":{"Handlers":{"/":{"Proxy":"http://127.0.0.1:4317"}}}}}`), nil
		}
		return []byte(`{}`), nil
	case strings.Contains(call, "tailscale serve --bg"):
		runner.configured = true
		return []byte("Available within your tailnet"), nil
	case strings.HasPrefix(call, "plutil "), strings.HasPrefix(call, "launchctl "):
		return nil, nil
	default:
		return nil, errors.New("unexpected command: " + call)
	}
}

func TestPhoneSetupIsIdempotentAndWritesLaunchAgent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("EVE_PHONE_HOME", home)
	t.Setenv("EVE_PHONE_CONFIG", filepath.Join(home, "phone.json"))
	t.Setenv("EVE_TAILSCALE_CLI", "/test/tailscale")
	t.Setenv("EVE_PHONE_SKIP_HEALTH", "1")
	t.Setenv("EVE_PHONE_SKIP_REMOTE_VERIFY", "1")
	runner := &fakePhoneRunner{}
	var stdout, stderr bytes.Buffer
	if code := runPhoneSetup(nil, runner, &stdout, &stderr); code != 0 {
		t.Fatalf("setup code = %d stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "EVE Approvals is ready") || !strings.Contains(stdout.String(), "https://mac.example.ts.net:8443/phone") {
		t.Fatalf("setup output = %q", stdout.String())
	}
	launchAgent, err := phoneLaunchAgentPath()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(launchAgent)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{phoneLaunchAgentLabel, "127.0.0.1:4317", "KeepAlive"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("LaunchAgent missing %q: %s", want, data)
		}
	}
	first, _ := defaultPhoneConfigStore().load()
	stdout.Reset()
	stderr.Reset()
	if code := runPhoneSetup(nil, runner, &stdout, &stderr); code != 0 {
		t.Fatalf("second setup code = %d stderr = %s", code, stderr.String())
	}
	second, _ := defaultPhoneConfigStore().load()
	if first.VAPIDPrivateKey != second.VAPIDPrivateKey {
		t.Fatal("idempotent setup rotated the VAPID key")
	}
}

func validPhoneSubscriptionInput(t *testing.T, endpoint string) registerPhoneSubscriptionInput {
	t.Helper()
	private, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	input := registerPhoneSubscriptionInput{Endpoint: endpoint}
	input.Keys.P256DH = base64.RawURLEncoding.EncodeToString(private.PublicKey().Bytes())
	input.Keys.Auth = base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{7}, 16))
	input.Device.Label = "Niko's iPhone"
	input.Device.UserAgent = "Mobile Safari"
	return input
}

func servePhoneRequest(t *testing.T, handler http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, "https://mac.example.ts.net:8443"+path, bytes.NewReader(body))
	request.Host = "mac.example.ts.net:8443"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(phoneTailscaleLoginHeader, "owner@example.com")
	request.Header.Set("X-Forwarded-Proto", "https")
	request.Header.Set("Origin", "https://mac.example.ts.net:8443")
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestPhoneDeliveryRemovesPermanentSubscription(t *testing.T) {
	pushServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusGone) }))
	defer pushServer.Close()
	store := &phoneConfigStore{path: filepath.Join(t.TempDir(), "phone.json")}
	input := validPhoneSubscriptionInput(t, pushServer.URL)
	_, err := store.update(func(config *phoneConfig) error {
		config.Enabled = true
		config.AllowedTailscaleLogin = "owner@example.com"
		if err := ensurePhoneVAPIDKey(config); err != nil {
			return err
		}
		config.Subscriptions = []phoneSubscription{{
			ID: "device", Endpoint: input.Endpoint, P256DH: input.Keys.P256DH, Auth: input.Keys.Auth, TailscaleUser: "owner@example.com",
		}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	manager := newPhoneManager(store, nil)
	manager.client = pushServer.Client()
	manager.retryDelays = []time.Duration{time.Millisecond, time.Millisecond}
	manager.deliverWithRetry(context.Background(), "plan:1", phoneNotificationPayload{Version: 1, Type: "plan_pending", URL: "/phone"}, phoneSubscription{
		ID: "device", Endpoint: input.Endpoint, P256DH: input.Keys.P256DH, Auth: input.Keys.Auth,
	})
	config, _ := store.load()
	if len(config.Subscriptions) != 0 {
		t.Fatalf("permanent subscription was not removed: %#v", config.Subscriptions)
	}
}

func TestPhoneDeliveryRetriesTransientFailures(t *testing.T) {
	attempts := 0
	pushServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer pushServer.Close()
	store := &phoneConfigStore{path: filepath.Join(t.TempDir(), "phone.json")}
	input := validPhoneSubscriptionInput(t, pushServer.URL)
	subscription := phoneSubscription{ID: "device", Endpoint: input.Endpoint, P256DH: input.Keys.P256DH, Auth: input.Keys.Auth, TailscaleUser: "owner@example.com"}
	_, err := store.update(func(config *phoneConfig) error {
		config.Enabled = true
		config.AllowedTailscaleLogin = "owner@example.com"
		if err := ensurePhoneVAPIDKey(config); err != nil {
			return err
		}
		config.Subscriptions = []phoneSubscription{subscription}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	manager := newPhoneManager(store, nil)
	manager.client = pushServer.Client()
	manager.retryDelays = []time.Duration{time.Millisecond, time.Millisecond}
	manager.deliverWithRetry(context.Background(), "plan:1", phoneNotificationPayload{Version: 1, Type: "plan_pending", URL: "/phone"}, subscription)
	config, err := store.load()
	if err != nil {
		t.Fatal(err)
	}
	delivery, found := config.delivery("plan:1", "device")
	if attempts != 3 || !found || delivery.State != "delivered" || delivery.Attempts != 3 {
		t.Fatalf("attempts=%d delivery=%#v found=%v", attempts, delivery, found)
	}
}
