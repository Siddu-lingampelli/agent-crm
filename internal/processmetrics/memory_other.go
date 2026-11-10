//go:build !windows && !linux

package processmetrics

import "errors"

func Memory() (MemorySnapshot, error) {
	return MemorySnapshot{}, errors.New("process resident memory is not supported on this platform")
}
