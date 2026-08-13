package common

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// VinSuffixLen is the last-4 window used in logs and audit (SPEC.md §8).
const VinSuffixLen = 4

// HashVIN is salted HMAC-SHA256 plus last 4 chars when longer than 4.
// Shorter values get an empty suffix so a rejected VIN is not stored (SPEC.md §8).
func HashVIN(salt, vin string) (hash string, suffix string) {
	mac := hmac.New(sha256.New, []byte(salt))
	_, _ = mac.Write([]byte(vin))
	return hex.EncodeToString(mac.Sum(nil)), VINSuffix(vin)
}

// VINSuffix is the last VinSuffixLen runes, or empty when the value is too short
// to be a VIN (rejected requests must not store a suffix that looks like one).
func VINSuffix(vin string) string {
	runes := []rune(vin)
	if len(runes) <= VinSuffixLen {
		return ""
	}
	return string(runes[len(runes)-VinSuffixLen:])
}
