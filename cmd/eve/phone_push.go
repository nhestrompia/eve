package main

import (
	"context"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/daaku/webpush"
	"github.com/rivo/uniseg"
)

const phonePushSubscriber = "https://github.com/nhestrompia/eve"

type phoneManager struct {
	store           *phoneConfigStore
	pendingPlans    func(context.Context) ([]*planRequest, error)
	client          *http.Client
	retryDelays     []time.Duration
	endpointAllowed func(string) bool
	logger          io.Writer
	reconcileMu     sync.Mutex
}

type phoneDeviceStatus struct {
	ID            string  `json:"id"`
	Label         string  `json:"label"`
	CreatedAt     string  `json:"createdAt"`
	LastSeenAt    string  `json:"lastSeenAt"`
	LastSuccessAt *string `json:"lastSuccessAt"`
	LastError     *string `json:"lastError"`
}

type phoneStatusResponse struct {
	Enabled          bool                `json:"enabled"`
	Origin           *string             `json:"origin"`
	TailscaleLogin   *string             `json:"tailscaleLogin"`
	VAPIDPublicKey   *string             `json:"vapidPublicKey"`
	PendingPlanCount int                 `json:"pendingPlanCount"`
	Devices          []phoneDeviceStatus `json:"devices"`
}

type registerPhoneSubscriptionInput struct {
	Endpoint       string `json:"endpoint"`
	ExpirationTime *int64 `json:"expirationTime"`
	Keys           struct {
		P256DH string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
	Device struct {
		Label     string `json:"label"`
		UserAgent string `json:"userAgent"`
	} `json:"device"`
}

type phoneNotificationPayload struct {
	Version       int    `json:"version"`
	Type          string `json:"type"`
	PlanRequestID string `json:"planRequestId,omitempty"`
	Revision      int    `json:"revision,omitempty"`
	Repository    string `json:"repository,omitempty"`
	Goal          string `json:"goal,omitempty"`
	URL           string `json:"url"`
	PendingCount  int    `json:"pendingCount"`
}

func newPhoneManager(store *phoneConfigStore, pending func(context.Context) ([]*planRequest, error)) *phoneManager {
	return &phoneManager{
		store:        store,
		pendingPlans: pending,
		client:       &http.Client{Timeout: 15 * time.Second},
		retryDelays:  []time.Duration{5 * time.Second, 30 * time.Second, 2 * time.Minute},
		endpointAllowed: func(host string) bool {
			host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
			return host == "push.apple.com" || strings.HasSuffix(host, ".push.apple.com")
		},
		logger: io.Discard,
	}
}

func (manager *phoneManager) status(ctx context.Context) (phoneStatusResponse, error) {
	config, err := manager.store.load()
	if errors.Is(err, os.ErrNotExist) {
		return phoneStatusResponse{Devices: []phoneDeviceStatus{}}, nil
	}
	if err != nil {
		return phoneStatusResponse{}, err
	}
	status := phoneStatusResponse{
		Enabled:        config.Enabled,
		Origin:         phoneOptionalString(config.origin()),
		TailscaleLogin: phoneOptionalString(config.AllowedTailscaleLogin),
		Devices:        make([]phoneDeviceStatus, 0, len(config.Subscriptions)),
	}
	if config.VAPIDPrivateKey != "" {
		publicKey, keyErr := phoneVAPIDPublicKey(config.VAPIDPrivateKey)
		err = keyErr
		if err != nil {
			return phoneStatusResponse{}, err
		}
		status.VAPIDPublicKey = &publicKey
	}
	if manager.pendingPlans != nil {
		plans, pendingErr := manager.pendingPlans(ctx)
		if pendingErr == nil {
			status.PendingPlanCount = len(plans)
		}
	}
	for _, subscription := range config.Subscriptions {
		status.Devices = append(status.Devices, sanitizePhoneSubscription(subscription))
	}
	sort.Slice(status.Devices, func(i, j int) bool { return status.Devices[i].CreatedAt < status.Devices[j].CreatedAt })
	return status, nil
}

func phoneVAPIDPublicKey(private string) (string, error) {
	key, err := webpush.ParseVAPIDKey(private)
	if err != nil {
		return "", fmt.Errorf("parse phone VAPID key: %w", err)
	}
	encoded := elliptic.Marshal(elliptic.P256(), key.PublicKey.X, key.PublicKey.Y)
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func sanitizePhoneSubscription(subscription phoneSubscription) phoneDeviceStatus {
	return phoneDeviceStatus{
		ID:            subscription.ID,
		Label:         subscription.Label,
		CreatedAt:     subscription.CreatedAt,
		LastSeenAt:    subscription.LastSeenAt,
		LastSuccessAt: phoneOptionalString(subscription.LastSuccessAt),
		LastError:     phoneOptionalString(subscription.LastError),
	}
}

func phoneOptionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func (manager *phoneManager) register(ctx context.Context, input registerPhoneSubscriptionInput, tailscaleUser string) (phoneDeviceStatus, error) {
	endpoint, err := validatePhoneSubscriptionInput(input, manager.endpointAllowed)
	if err != nil {
		return phoneDeviceStatus{}, err
	}
	pending := []*planRequest{}
	if manager.pendingPlans != nil {
		pending, err = manager.pendingPlans(ctx)
		if err != nil {
			return phoneDeviceStatus{}, fmt.Errorf("read pending plans: %w", err)
		}
	}
	id := phoneSubscriptionID(endpoint)
	now := nowUTC()
	var result phoneSubscription
	_, err = manager.store.update(func(config *phoneConfig) error {
		if !config.Enabled {
			return errors.New("phone approvals are disabled; run `eve phone setup`")
		}
		if !strings.EqualFold(strings.TrimSpace(config.AllowedTailscaleLogin), strings.TrimSpace(tailscaleUser)) {
			return errors.New("Tailscale identity does not match the configured phone user")
		}
		result = phoneSubscription{
			ID:            id,
			Endpoint:      endpoint,
			P256DH:        strings.TrimSpace(input.Keys.P256DH),
			Auth:          strings.TrimSpace(input.Keys.Auth),
			ExpirationMs:  input.ExpirationTime,
			Label:         boundedPhoneText(input.Device.Label, 80, "iPhone"),
			UserAgent:     boundedPhoneText(input.Device.UserAgent, 512, ""),
			TailscaleUser: config.AllowedTailscaleLogin,
			CreatedAt:     now,
			LastSeenAt:    now,
		}
		for index := range config.Subscriptions {
			if config.Subscriptions[index].ID != id {
				continue
			}
			result.CreatedAt = config.Subscriptions[index].CreatedAt
			result.LastSuccessAt = config.Subscriptions[index].LastSuccessAt
			config.Subscriptions[index] = result
			seedExistingPhoneDeliveries(config, pending, id)
			return nil
		}
		config.Subscriptions = append(config.Subscriptions, result)
		seedExistingPhoneDeliveries(config, pending, id)
		return nil
	})
	return sanitizePhoneSubscription(result), err
}

func seedExistingPhoneDeliveries(config *phoneConfig, plans []*planRequest, subscriptionID string) {
	for _, plan := range plans {
		upsertPhoneDelivery(config, phoneDelivery{
			NotificationKey: phoneNotificationKey(plan),
			SubscriptionID:  subscriptionID,
			State:           "skipped_existing",
			LastAttemptAt:   nowUTC(),
		})
	}
}

func validatePhoneSubscriptionInput(input registerPhoneSubscriptionInput, endpointAllowed func(string) bool) (string, error) {
	endpoint := strings.TrimSpace(input.Endpoint)
	if len(endpoint) == 0 || len(endpoint) > 2048 {
		return "", errors.New("push endpoint is required and must be at most 2048 characters")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
		return "", errors.New("push endpoint must be an HTTPS URL")
	}
	if endpointAllowed == nil || !endpointAllowed(parsed.Hostname()) {
		return "", errors.New("push endpoint must use Apple's push service")
	}
	p256dh, err := decodePhoneBase64(input.Keys.P256DH)
	if err != nil || len(p256dh) != 65 || p256dh[0] != 4 {
		return "", errors.New("p256dh must be a valid P-256 public key")
	}
	auth, err := decodePhoneBase64(input.Keys.Auth)
	if err != nil || len(auth) < 16 || len(auth) > 64 {
		return "", errors.New("auth must be a valid Web Push authentication secret")
	}
	if len(input.Device.Label) > 80 || len(input.Device.UserAgent) > 512 {
		return "", errors.New("device metadata is too long")
	}
	return parsed.String(), nil
}

func decodePhoneBase64(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	for _, encoding := range []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.StdEncoding} {
		if decoded, err := encoding.DecodeString(value); err == nil {
			return decoded, nil
		}
	}
	return nil, errors.New("invalid base64")
}

func boundedPhoneText(value string, maximum int, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	runes := []rune(value)
	if len(runes) > maximum {
		return string(runes[:maximum])
	}
	return value
}

func (manager *phoneManager) removeSubscription(id, tailscaleUser string) error {
	_, err := manager.store.update(func(config *phoneConfig) error {
		if !strings.EqualFold(strings.TrimSpace(config.AllowedTailscaleLogin), strings.TrimSpace(tailscaleUser)) {
			return errors.New("Tailscale identity does not match the configured phone user")
		}
		subscriptions := config.Subscriptions[:0]
		for _, subscription := range config.Subscriptions {
			if subscription.ID != id {
				subscriptions = append(subscriptions, subscription)
			}
		}
		config.Subscriptions = subscriptions
		deliveries := config.Deliveries[:0]
		for _, delivery := range config.Deliveries {
			if delivery.SubscriptionID != id {
				deliveries = append(deliveries, delivery)
			}
		}
		config.Deliveries = deliveries
		return nil
	})
	return err
}

func (manager *phoneManager) sendTest(ctx context.Context, subscriptionID string) error {
	config, err := manager.store.load()
	if err != nil {
		return err
	}
	var subscription *phoneSubscription
	for index := range config.Subscriptions {
		if config.Subscriptions[index].ID == subscriptionID {
			copy := config.Subscriptions[index]
			subscription = &copy
			break
		}
	}
	if subscription == nil {
		return errors.New("phone subscription not found")
	}
	payload := phoneNotificationPayload{Version: 1, Type: "connected", Goal: "Notifications are ready", URL: "/phone"}
	err = manager.send(ctx, config, *subscription, payload, phoneTopic("test:"+subscriptionID))
	manager.recordSubscriptionResult(subscriptionID, err)
	return err
}

func (manager *phoneManager) run(ctx context.Context, events *runtimeEvents) {
	if events == nil {
		return
	}
	stream, unsubscribe := events.subscribe()
	defer unsubscribe()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	var debounce *time.Timer
	var debounceC <-chan time.Time
	manager.reconcile(ctx)
	for {
		select {
		case <-ctx.Done():
			if debounce != nil {
				debounce.Stop()
			}
			return
		case <-ticker.C:
			manager.reconcile(ctx)
		case event := <-stream:
			if event.Kind == runtimeEventPlans || event.Kind == runtimeEventAll {
				if debounce == nil {
					debounce = time.NewTimer(250 * time.Millisecond)
				} else {
					if !debounce.Stop() {
						select {
						case <-debounce.C:
						default:
						}
					}
					debounce.Reset(250 * time.Millisecond)
				}
				debounceC = debounce.C
			}
		case <-debounceC:
			debounceC = nil
			manager.reconcile(ctx)
		}
	}
}

func (manager *phoneManager) reconcile(ctx context.Context) {
	manager.reconcileMu.Lock()
	defer manager.reconcileMu.Unlock()
	config, err := manager.store.load()
	if err != nil || !config.Enabled || len(config.Subscriptions) == 0 || manager.pendingPlans == nil {
		return
	}
	plans, err := manager.pendingPlans(ctx)
	if err != nil {
		fmt.Fprintf(manager.logger, "phone notification reconciliation: %v\n", err)
		return
	}
	for _, plan := range plans {
		key := phoneNotificationKey(plan)
		payload := phonePayloadForPlan(plan, len(plans))
		for _, subscription := range config.Subscriptions {
			if delivery, found := config.delivery(key, subscription.ID); found && (delivery.State == "delivered" || delivery.State == "skipped_existing") {
				continue
			}
			manager.deliverWithRetry(ctx, key, payload, subscription)
		}
	}
}

func (manager *phoneManager) deliverWithRetry(ctx context.Context, key string, payload phoneNotificationPayload, subscription phoneSubscription) {
	for attempt := 1; attempt <= 3; attempt++ {
		config, err := manager.store.load()
		if err != nil || !config.Enabled {
			return
		}
		err = manager.send(ctx, config, subscription, payload, phoneTopic(key))
		state := "failed"
		if err == nil {
			state = "delivered"
		}
		manager.recordDelivery(key, subscription.ID, state, attempt, err)
		if err == nil {
			return
		}
		var pushError *webpush.Error
		if errors.As(err, &pushError) && pushError.Permanent {
			_ = manager.removeSubscription(subscription.ID, config.AllowedTailscaleLogin)
			return
		}
		if attempt == 3 {
			return
		}
		delay := manager.retryDelays[attempt-1]
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}
}

func (manager *phoneManager) send(ctx context.Context, config phoneConfig, subscription phoneSubscription, payload phoneNotificationPayload, topic string) error {
	privateKey, err := webpush.ParseVAPIDKey(config.VAPIDPrivateKey)
	if err != nil {
		return fmt.Errorf("parse VAPID key: %w", err)
	}
	message, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return webpush.Send(ctx, message, &webpush.Subscription{
		Endpoint: subscription.Endpoint,
		Keys:     webpush.Keys{Auth: subscription.Auth, P256dh: subscription.P256DH},
	}, &webpush.Config{
		Client:     manager.client,
		VAPIDKey:   privateKey,
		Subscriber: phonePushSubscriber,
		TTL:        24 * time.Hour,
		Topic:      topic,
		Urgency:    webpush.UrgencyHigh,
	})
}

func (manager *phoneManager) recordDelivery(key, subscriptionID, state string, attempts int, deliveryErr error) {
	_, _ = manager.store.update(func(config *phoneConfig) error {
		upsertPhoneDelivery(config, phoneDelivery{
			NotificationKey: key,
			SubscriptionID:  subscriptionID,
			State:           state,
			Attempts:        attempts,
			LastAttemptAt:   nowUTC(),
		})
		for index := range config.Subscriptions {
			if config.Subscriptions[index].ID != subscriptionID {
				continue
			}
			if deliveryErr == nil {
				config.Subscriptions[index].LastSuccessAt = nowUTC()
				config.Subscriptions[index].LastError = ""
				config.Subscriptions[index].FailureCount = 0
			} else {
				config.Subscriptions[index].LastError = deliveryErr.Error()
				config.Subscriptions[index].FailureCount++
			}
		}
		return nil
	})
	if deliveryErr != nil {
		fmt.Fprintf(manager.logger, "phone push plan=%s subscription=%s attempt=%d error=%v\n", key, subscriptionID, attempts, deliveryErr)
	} else {
		fmt.Fprintf(manager.logger, "phone push plan=%s subscription=%s attempt=%d result=delivered\n", key, subscriptionID, attempts)
	}
}

func (manager *phoneManager) recordSubscriptionResult(subscriptionID string, deliveryErr error) {
	_, _ = manager.store.update(func(config *phoneConfig) error {
		for index := range config.Subscriptions {
			if config.Subscriptions[index].ID != subscriptionID {
				continue
			}
			if deliveryErr == nil {
				config.Subscriptions[index].LastSuccessAt = nowUTC()
				config.Subscriptions[index].LastError = ""
				config.Subscriptions[index].FailureCount = 0
			} else {
				config.Subscriptions[index].LastError = deliveryErr.Error()
				config.Subscriptions[index].FailureCount++
			}
		}
		return nil
	})
}

func phoneNotificationKey(plan *planRequest) string {
	return fmt.Sprintf("%s:%d", plan.PlanRequestID, plan.CurrentRevision)
}

func phonePayloadForPlan(plan *planRequest, pendingCount int) phoneNotificationPayload {
	goal := "Plan awaiting review"
	for _, revision := range plan.Revisions {
		if revision.Revision == plan.CurrentRevision {
			goal = truncatePhoneGoal(revision.Goal, 120)
			break
		}
	}
	return phoneNotificationPayload{
		Version:       1,
		Type:          "plan_pending",
		PlanRequestID: plan.PlanRequestID,
		Revision:      plan.CurrentRevision,
		Repository:    plan.Repository,
		Goal:          goal,
		URL:           "/phone/plans/" + url.PathEscape(plan.PlanRequestID),
		PendingCount:  pendingCount,
	}
}

func truncatePhoneGoal(value string, maximum int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Plan awaiting review"
	}
	graphemes := uniseg.NewGraphemes(value)
	var builder strings.Builder
	count := 0
	for graphemes.Next() {
		if count == maximum {
			return strings.TrimSpace(builder.String()) + "…"
		}
		builder.WriteString(graphemes.Str())
		count++
	}
	return builder.String()
}

func phoneTopic(value string) string {
	sum := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(sum[:])[:32]
}
