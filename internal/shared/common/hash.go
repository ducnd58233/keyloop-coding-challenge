package common

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

const vinSuffixLen = 4

// HashVIN is salted HMAC-SHA256 plus last 4 chars when longer than 4.
// Shorter values get an empty suffix so a rejected VIN is not stored (SPEC.md §8).
func HashVIN(salt, vin string) (hash string, suffix string) {
	mac := hmac.New(sha256.New, []byte(salt))
	_, _ = mac.Write([]byte(vin))
	return hex.EncodeToString(mac.Sum(nil)), vinSuffix(vin)
}

func vinSuffix(vin string) string {
	runes := []rune(vin)
	if len(runes) <= vinSuffixLen {
		return ""
	}
	return string(runes[len(runes)-vinSuffixLen:])
}
