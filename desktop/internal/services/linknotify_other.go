//go:build !windows

package services

// Other platforms fall back to the poll loop in watchLiveLinks.
func watchLinkNotifications() (<-chan struct{}, func()) {
	return make(chan struct{}), func() {}
}
