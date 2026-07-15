package credentialcipher

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

const envelopePrefix = "enc:v1:"

type Key struct {
	Version string
	Secret  string
}

type Box struct {
	currentVersion string
	keys           map[string][]byte
	legacyOrder    []string
}

func New(current Key, previous []Key) (*Box, error) {
	current.Version = normalizeVersion(current.Version)
	current.Secret = strings.TrimSpace(current.Secret)
	if current.Version == "" || current.Secret == "" {
		return nil, errors.New("current credential key version and secret are required")
	}
	box := &Box{
		currentVersion: current.Version,
		keys:           map[string][]byte{},
		legacyOrder:    []string{current.Version},
	}
	box.keys[current.Version] = deriveKey(current.Secret)
	for _, item := range previous {
		version := normalizeVersion(item.Version)
		secret := strings.TrimSpace(item.Secret)
		if version == "" || secret == "" {
			return nil, errors.New("previous credential key version and secret are required")
		}
		if _, exists := box.keys[version]; exists {
			return nil, fmt.Errorf("duplicate credential key version %q", version)
		}
		box.keys[version] = deriveKey(secret)
		box.legacyOrder = append(box.legacyOrder, version)
	}
	return box, nil
}

func ParsePreviousKeys(raw string) ([]Key, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	items := make([]Key, 0)
	for _, entry := range strings.Split(raw, ",") {
		parts := strings.SplitN(strings.TrimSpace(entry), "=", 2)
		if len(parts) != 2 {
			return nil, errors.New("CREDENTIALS_PREVIOUS_KEYS must use version=secret entries")
		}
		items = append(items, Key{Version: parts[0], Secret: parts[1]})
	}
	return items, nil
}

func (b *Box) Encrypt(plain string) (string, error) {
	if b == nil {
		return "", errors.New("credential cipher is not configured")
	}
	version := b.currentVersion
	payload, err := seal(b.keys[version], []byte(plain), []byte(envelopePrefix+version))
	if err != nil {
		return "", err
	}
	return envelopePrefix + version + ":" + base64.RawStdEncoding.EncodeToString(payload), nil
}

func (b *Box) Decrypt(ciphertext string) (string, error) {
	if b == nil {
		return "", errors.New("credential cipher is not configured")
	}
	if version, payload, ok, err := parseEnvelope(ciphertext); ok || err != nil {
		if err != nil {
			return "", err
		}
		key, exists := b.keys[version]
		if !exists {
			return "", fmt.Errorf("credential key version %q is not configured", version)
		}
		plain, err := open(key, payload, []byte(envelopePrefix+version))
		return string(plain), err
	}

	payload, err := base64.StdEncoding.DecodeString(strings.TrimSpace(ciphertext))
	if err != nil {
		return "", fmt.Errorf("decode legacy credential ciphertext: %w", err)
	}
	var failures []string
	for _, version := range b.legacyOrder {
		plain, openErr := open(b.keys[version], payload, nil)
		if openErr == nil {
			return string(plain), nil
		}
		failures = append(failures, version)
	}
	sort.Strings(failures)
	return "", fmt.Errorf("decrypt legacy credential ciphertext with configured keys %v: authentication failed", failures)
}

func (b *Box) NeedsReencrypt(ciphertext string) bool {
	version, _, ok, err := parseEnvelope(ciphertext)
	return err != nil || !ok || version != b.currentVersion
}

func (b *Box) Reencrypt(ciphertext string) (string, bool, error) {
	if !b.NeedsReencrypt(ciphertext) {
		return ciphertext, false, nil
	}
	plain, err := b.Decrypt(ciphertext)
	if err != nil {
		return "", false, err
	}
	rotated, err := b.Encrypt(plain)
	if err != nil {
		return "", false, err
	}
	return rotated, true, nil
}

func deriveKey(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

func seal(key, plain, additionalData []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate credential nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plain, additionalData), nil
}

func open(key, payload, additionalData []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(payload) < gcm.NonceSize() {
		return nil, errors.New("credential ciphertext is too short")
	}
	nonce := payload[:gcm.NonceSize()]
	plain, err := gcm.Open(nil, nonce, payload[gcm.NonceSize():], additionalData)
	if err != nil {
		return nil, fmt.Errorf("decrypt credential ciphertext: %w", err)
	}
	return plain, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create credential cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create credential GCM: %w", err)
	}
	return gcm, nil
}

func parseEnvelope(ciphertext string) (string, []byte, bool, error) {
	value := strings.TrimSpace(ciphertext)
	if !strings.HasPrefix(value, envelopePrefix) {
		return "", nil, false, nil
	}
	remainder := strings.TrimPrefix(value, envelopePrefix)
	parts := strings.SplitN(remainder, ":", 2)
	if len(parts) != 2 || normalizeVersion(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", nil, true, errors.New("credential ciphertext envelope is invalid")
	}
	payload, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", nil, true, fmt.Errorf("decode credential ciphertext: %w", err)
	}
	return normalizeVersion(parts[0]), payload, true, nil
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" || strings.ContainsAny(version, ":=, \t\r\n") {
		return ""
	}
	return version
}
