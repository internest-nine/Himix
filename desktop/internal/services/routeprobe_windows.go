//go:build windows

package services

import (
	"fmt"
	"math/bits"
	"net"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// IP_UNICAST_IF pins an outbound connection to one interface regardless of what
// the route table would have chosen. That is what makes a probe harmless: a
// wrong candidate fails with a timeout instead of leaking traffic, and no route
// has to be created or removed.
const routeProbeUnicastIF = 31

func probeAdapterConnect(adapter AdapterView, address string, port int, timeout time.Duration) (time.Duration, error) {
	source := net.ParseIP(adapter.Address)
	if source == nil || source.To4() == nil {
		return 0, fmt.Errorf("网卡 %s 没有 IPv4 地址", adapter.Name)
	}
	dialer := &net.Dialer{Timeout: timeout, LocalAddr: &net.TCPAddr{IP: source.To4()}}
	if adapter.IfIndex > 0 {
		ifIndex := adapter.IfIndex
		// IPv4 expects IP_UNICAST_IF in network byte order.
		networkOrderIndex := int(bits.ReverseBytes32(uint32(ifIndex)))
		dialer.Control = func(_, _ string, raw syscall.RawConn) error {
			var controlErr error
			if err := raw.Control(func(handle uintptr) {
				controlErr = windows.SetsockoptInt(
					windows.Handle(handle), windows.IPPROTO_IP,
					routeProbeUnicastIF, networkOrderIndex,
				)
			}); err != nil {
				return err
			}
			return controlErr
		}
	}
	start := time.Now()
	conn, err := dialer.Dial("tcp4", net.JoinHostPort(address, strconv.Itoa(port)))
	if err != nil {
		return 0, err
	}
	elapsed := time.Since(start)
	_ = conn.Close()
	return elapsed, nil
}
