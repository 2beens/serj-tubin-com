package pkg

import "unsafe"

// BytesToString converts bytes slice to a string without extra allocation
func BytesToString(buf []byte) string {
	return *(*string)(unsafe.Pointer(&buf))
}
