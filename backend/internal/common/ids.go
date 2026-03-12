package common

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func NewPrefixedID(prefix string) (string, error) {
	randomBytes := make([]byte, 6)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}

	return fmt.Sprintf("%s_%d_%s", prefix, time.Now().UnixMilli(), hex.EncodeToString(randomBytes)), nil
}
