package pkg

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"unsafe"
)

// BytesToString converts bytes slice to a string without extra allocation
func BytesToString(buf []byte) string {
	return *(*string)(unsafe.Pointer(&buf))
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
func GenerateRandomString(s int) (string, error) {
	b, err := GenerateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}

// DirectoryExists returns whether the given file or directory exists
func PathExists(path string, isDir bool) (bool, error) {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if (isDir && stat.IsDir()) || (!isDir && !stat.IsDir()) {
		return true, nil
	}
	return false, err
}
