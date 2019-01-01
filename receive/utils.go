package receive

import "unsafe"

// ToString ToString
func ToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
