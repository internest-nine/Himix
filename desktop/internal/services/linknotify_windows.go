//go:build windows

package services

import (
	"syscall"
	"unsafe"
)

var (
	procNotifyIpInterfaceChange = iphlpapiDLL.NewProc("NotifyIpInterfaceChange")
	procCancelMibChangeNotify2  = iphlpapiDLL.NewProc("CancelMibChangeNotify2")
)

// AF_UNSPEC asks for every address family in one subscription.
const linkNotifyFamilyUnspecified = 0

// watchLinkNotifications turns Windows' interface-change notifications into a
// single edge-triggered signal. Disabling a Wi-Fi adapter, unplugging a cable or
// losing the DHCP lease all raise it, which is what makes losing a manually
// disabled link feel immediate instead of waiting for the next poll.
//
// The callback runs on a system thread and must stay trivial: it only signals a
// one-deep channel, and the caller debounces the burst that a single action
// produces. Returning early keeps the notification pump from backing up.
func watchLinkNotifications() (<-chan struct{}, func()) {
	changes := make(chan struct{}, 1)
	callback := syscall.NewCallback(func(_, _, _ uintptr) uintptr {
		select {
		case changes <- struct{}{}:
		default:
		}
		return 0
	})
	var handle syscall.Handle
	result, _, callErr := procNotifyIpInterfaceChange.Call(
		linkNotifyFamilyUnspecified,
		callback,
		0,
		0, // InitialNotification: the poll loop already has the current state.
		uintptr(unsafe.Pointer(&handle)),
	)
	if result != 0 {
		_ = callErr
		return changes, func() {}
	}
	return changes, func() {
		if handle != 0 {
			procCancelMibChangeNotify2.Call(uintptr(handle))
		}
	}
}
