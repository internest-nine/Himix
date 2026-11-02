package services

import (
	"reflect"
	"testing"
)

func TestRouteProbeHostNormalisesPastableInput(t *testing.T) {
	cases := []struct{ input, want string }{
		{"portal.example.com", "portal.example.com"},
		{"https://portal.example.com/pages/home.aspx", "portal.example.com"},
		{"http://203.0.113.8:8081/portal/login.jsp?trid=1", "203.0.113.8"},
		{"https://www.example.com/item/123?id=5", "www.example.com"},
		{"  HTTPS://Example.COM:8443/path  ", "example.com"},
		{"example.com.", "example.com"},
		{"1.1.1.1", "1.1.1.1"},
		{"https://user:pw@intranet.example.com/x", "intranet.example.com"},
		{"", ""},
	}
	for _, testCase := range cases {
		if got := RouteProbeHost(testCase.input); got != testCase.want {
			t.Errorf("RouteProbeHost(%q) = %q, want %q", testCase.input, got, testCase.want)
		}
	}
}

func TestRouteProbeAddressesKeepsLiteralIPWithoutDNS(t *testing.T) {
	if got := routeProbeAddresses("10.177.101.18"); !reflect.DeepEqual(got, []string{"10.177.101.18"}) {
		t.Fatalf("literal address probe = %v", got)
	}
}

func TestRouteProbeCandidatesSkipVirtualAndNonPhysical(t *testing.T) {
	adapters := []AdapterView{
		{ID: "a", Name: "以太网", Kind: "ethernet", Address: "10.176.8.14", Operational: true},
		{ID: "b", Name: "WLAN", Kind: "wifi", Address: "192.168.0.9", Operational: true},
		{ID: "c", Name: "vEthernet (Default Switch)", Kind: "ethernet", Address: "172.20.64.1", Operational: true, IsVirtual: true},
		{ID: "d", Name: "VMware VMnet8", Kind: "ethernet", Address: "192.168.11.1", Operational: true, IsVirtual: true},
		{ID: "e", Name: "Himix-Tun", Kind: "ethernet", Address: "10.255.255.1", Operational: true, IsVirtual: true},
		{ID: "f", Name: "断开的有线", Kind: "ethernet", Operational: false},
	}
	candidates := routeProbeCandidates(adapters)
	var names []string
	for _, candidate := range candidates {
		names = append(names, candidate.Name)
	}
	want := []string{"以太网", "WLAN"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("candidates = %v, want %v", names, want)
	}
}
