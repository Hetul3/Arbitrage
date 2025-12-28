package hashutil

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashStrings returns a SHA256 hash of the provided strings with newline separators.
func HashStrings(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))
}
