package pkg

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"time"
	"unsafe"
)

// BytesToString converts bytes slice to a string without extra allocation
func BytesToString(buf []byte) string {
	return *(*string)(unsafe.Pointer(&buf))
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateRandomString generates a random string of the specified length
// containing only alphanumeric characters.
func GenerateRandomString(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("string length must be a positive number")
	}
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(
			rand.Reader,
			big.NewInt(int64(len(charset))),
		)
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}

// PathExists returns whether the given file or directory exists
func PathExists(path string, isDir bool) (bool, error) {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", path, err)
	}
	if (isDir && stat.IsDir()) || (!isDir && !stat.IsDir()) {
		return true, nil
	}
	return false, nil
}

func Compress(src string, buf io.Writer) error {
	// tar > gzip > buf
	gzipWriter := gzip.NewWriter(buf)
	tarWriter := tar.NewWriter(gzipWriter)

	// walk through every file in the folder
	if walkErr := filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// generate tar header
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		// must provide real name
		// (see https://golang.org/src/archive/tar/common.go?#L626)
		relPath, err := filepath.Rel(src, file)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		// write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// if not a dir, write file content
		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tarWriter, data); err != nil {
				return err
			}
		}

		return nil
	}); walkErr != nil {
		return walkErr
	}

	// produce tar
	if err := tarWriter.Close(); err != nil {
		return err
	}
	// produce gzip
	if err := gzipWriter.Close(); err != nil {
		return err
	}

	return nil
}

// SleepWithContext sleeps for the specified duration or until the context is canceled/done.
func SleepWithContext(ctx context.Context, duration time.Duration) {
	select {
	case <-time.After(duration):
		return
	case <-ctx.Done():
		return
	}
}
