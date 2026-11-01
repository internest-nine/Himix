//go:build !windows

package services

import (
	"errors"
	"time"
)

func probeAdapterConnect(adapter AdapterView, address string, port int, timeout time.Duration) (time.Duration, error) {
	return 0, errors.New("出口探测仅支持 Windows")
}
