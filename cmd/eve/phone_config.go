package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/daaku/webpush"
)

const phoneConfigSchemaVersion = 1

type phoneConfig struct {
	SchemaVersion         int                 `json:"schemaVersion"`
	Enabled               bool                `json:"enabled"`
	RuntimeAddr           string              `json:"runtimeAddr"`
	ServeHost             string              `json:"serveHost"`
	ServePort             int                 `json:"servePort"`
	AllowedTailscaleLogin string              `json:"allowedTailscaleLogin"`
	VAPIDPrivateKey       string              `json:"vapidPrivateKey"`
	Subscriptions         []phoneSubscription `json:"subscriptions"`
	Deliveries            []phoneDelivery     `json:"deliveries"`
	UpdatedAt             string              `json:"updatedAt"`
}

type phoneSubscription struct {
	ID            string `json:"id"`
	Endpoint      string `json:"endpoint"`
	P256DH        string `json:"p256dh"`
	Auth          string `json:"auth"`
	ExpirationMs  *int64 `json:"expirationMs,omitempty"`
	Label         string `json:"label"`
	UserAgent     string `json:"userAgent"`
	TailscaleUser string `json:"tailscaleUser"`
	CreatedAt     string `json:"createdAt"`
	LastSeenAt    string `json:"lastSeenAt"`
	LastSuccessAt string `json:"lastSuccessAt,omitempty"`
	LastError     string `json:"lastError,omitempty"`
	FailureCount  int    `json:"failureCount"`
}

type phoneDelivery struct {
	NotificationKey string `json:"notificationKey"`
	SubscriptionID  string `json:"subscriptionId"`
	State           string `json:"state"`
	Attempts        int    `json:"attempts"`
	LastAttemptAt   string `json:"lastAttemptAt"`
}

type phoneConfigStore struct {
	mu   sync.Mutex
	path string
}

func defaultPhoneConfigStore() *phoneConfigStore {
	return &phoneConfigStore{path: phoneConfigPath()}
}

func phoneConfigPath() string {
	if configured := strings.TrimSpace(os.Getenv("EVE_PHONE_CONFIG")); configured != "" {
		if absolute, err := filepath.Abs(configured); err == nil {
			return absolute
		}
		return configured
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "eve", "phone.json")
	}
	return filepath.Join(configDir, "eve", "phone.json")
}

func (store *phoneConfigStore) load() (phoneConfig, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.loadLocked()
}

func (store *phoneConfigStore) loadLocked() (phoneConfig, error) {
	data, err := os.ReadFile(store.path)
	if err != nil {
		return phoneConfig{}, err
	}
	var config phoneConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return phoneConfig{}, fmt.Errorf("parse phone configuration: %w", err)
	}
	if config.SchemaVersion != phoneConfigSchemaVersion {
		return phoneConfig{}, fmt.Errorf("phone configuration schemaVersion is %d; expected %d", config.SchemaVersion, phoneConfigSchemaVersion)
	}
	if config.Subscriptions == nil {
		config.Subscriptions = []phoneSubscription{}
	}
	if config.Deliveries == nil {
		config.Deliveries = []phoneDelivery{}
	}
	return config, nil
}

func (store *phoneConfigStore) update(update func(*phoneConfig) error) (phoneConfig, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	config, err := store.loadLocked()
	if errors.Is(err, os.ErrNotExist) {
		config = phoneConfig{
			SchemaVersion: phoneConfigSchemaVersion,
			RuntimeAddr:   "127.0.0.1:4317",
			ServePort:     8443,
			Subscriptions: []phoneSubscription{},
			Deliveries:    []phoneDelivery{},
		}
	} else if err != nil {
		return phoneConfig{}, err
	}
	if err := update(&config); err != nil {
		return phoneConfig{}, err
	}
	config.SchemaVersion = phoneConfigSchemaVersion
	config.UpdatedAt = nowUTC()
	if config.Subscriptions == nil {
		config.Subscriptions = []phoneSubscription{}
	}
	if config.Deliveries == nil {
		config.Deliveries = []phoneDelivery{}
	}
	if err := store.writeLocked(config); err != nil {
		return phoneConfig{}, err
	}
	return config, nil
}

func (store *phoneConfigStore) writeLocked(config phoneConfig) error {
	if err := os.MkdirAll(filepath.Dir(store.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(store.path), ".phone-*.json")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(append(data, '\n')); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, store.path); err != nil {
		return err
	}
	return os.Chmod(store.path, 0o600)
}

func ensurePhoneVAPIDKey(config *phoneConfig) error {
	if strings.TrimSpace(config.VAPIDPrivateKey) != "" {
		_, err := webpush.ParseVAPIDKey(config.VAPIDPrivateKey)
		return err
	}
	key, err := webpush.GenerateVAPIDKey()
	if err != nil {
		return err
	}
	config.VAPIDPrivateKey = key
	return nil
}

func phoneSubscriptionID(endpoint string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(endpoint)))
	return hex.EncodeToString(sum[:12])
}

func (config phoneConfig) origin() string {
	host := strings.TrimSpace(strings.TrimSuffix(config.ServeHost, "."))
	if host == "" || config.ServePort <= 0 {
		return ""
	}
	return fmt.Sprintf("https://%s:%d", host, config.ServePort)
}

func (config phoneConfig) delivery(notificationKey, subscriptionID string) (phoneDelivery, bool) {
	for _, delivery := range config.Deliveries {
		if delivery.NotificationKey == notificationKey && delivery.SubscriptionID == subscriptionID {
			return delivery, true
		}
	}
	return phoneDelivery{}, false
}

func upsertPhoneDelivery(config *phoneConfig, next phoneDelivery) {
	for index := range config.Deliveries {
		if config.Deliveries[index].NotificationKey == next.NotificationKey && config.Deliveries[index].SubscriptionID == next.SubscriptionID {
			config.Deliveries[index] = next
			return
		}
	}
	config.Deliveries = append(config.Deliveries, next)
}
