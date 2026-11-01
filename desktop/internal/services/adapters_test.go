package services

import "testing"

func TestIsHimixManagedAdapter(t *testing.T) {
	tests := map[string]bool{
		"Himix-Tun":   true,
		" himix-tun ": true,
		"WLAN":          false,
		"Other TUN":     false,
	}
	for name, expected := range tests {
		if actual := isHimixManagedAdapter(name); actual != expected {
			t.Fatalf("isHimixManagedAdapter(%q) = %v; want %v", name, actual, expected)
		}
	}
}
