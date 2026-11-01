package services

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RouteProbeHop is one adapter's attempt to reach the entered target.
type RouteProbeHop struct {
	AdapterID     string `json:"adapter_id"`
	AdapterName   string `json:"adapter_name"`
	Kind          string `json:"kind"`
	SourceAddress string `json:"source_address"`
	Target        string `json:"target"`
	OK            bool   `json:"ok"`
	LatencyMS     int64  `json:"latency_ms"`
	Detail        string `json:"detail,omitempty"`
}

type RouteProbeReport struct {
	Value                string          `json:"value"`
	Host                 string          `json:"host"`
	Addresses            []string        `json:"addresses"`
	Ports                []int           `json:"ports"`
	Hops                 []RouteProbeHop `json:"hops"`
	RecommendedAdapterID string          `json:"recommended_adapter_id,omitempty"`
	RecommendedName      string          `json:"recommended_name,omitempty"`
	AllFailed            bool            `json:"all_failed"`
}

type RouteProbeService struct {
	adapters *AdapterService
}

func NewRouteProbeService(adapters *AdapterService) *RouteProbeService {
	return &RouteProbeService{adapters: adapters}
}

// Both ports are tried because an internal portal is often plain HTTP while a
// public site answers on 443; a successful TCP handshake is enough to prove the
// path exists, so a 401/403 answer still counts. A port typed by the user is
// always tried first — an internal app on :8081 would otherwise look dead.
var routeProbePorts = []int{443, 80}

const (
	routeProbeResolveTimeout = 3 * time.Second
	routeProbeDialTimeout    = 2500 * time.Millisecond
	routeProbeMaxAddresses   = 2
)

var routeProbeScheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.\-]*://`)

// Probe answers one question: which of this machine's physical adapters can
// actually reach the entered target? The route table is never modified — every
// attempt is pinned to one adapter, so a wrong candidate costs a timeout and
// nothing else.
func (s *RouteProbeService) Probe(value string) (RouteProbeReport, error) {
	report := RouteProbeReport{Value: strings.TrimSpace(value)}
	host := RouteProbeHost(report.Value)
	if host == "" {
		return report, fmt.Errorf("无法识别目标地址")
	}
	report.Host = host
	report.Ports = routeProbePortsFor(report.Value)
	report.Addresses = routeProbeAddresses(host)
	if len(report.Addresses) == 0 {
		return report, fmt.Errorf("域名 %s 解析不到 IP", host)
	}

	adapters, err := s.adapters.List()
	if err != nil {
		return report, fmt.Errorf("读取网卡失败：%w", err)
	}
	candidates := routeProbeCandidates(adapters)
	if len(candidates) == 0 {
		return report, fmt.Errorf("没有可用于测试的物理网卡")
	}

	hops := make([]RouteProbeHop, len(candidates))
	var wg sync.WaitGroup
	for i, adapter := range candidates {
		wg.Add(1)
		go func(index int, adapter AdapterView) {
			defer wg.Done()
			hops[index] = probeAdapterTarget(adapter, report.Addresses, report.Ports)
		}(i, adapter)
	}
	wg.Wait()

	sort.SliceStable(hops, func(i, j int) bool {
		if hops[i].OK != hops[j].OK {
			return hops[i].OK
		}
		return hops[i].LatencyMS < hops[j].LatencyMS
	})
	report.Hops = hops
	for _, hop := range hops {
		if hop.OK {
			report.RecommendedAdapterID = hop.AdapterID
			report.RecommendedName = hop.AdapterName
			break
		}
	}
	report.AllFailed = report.RecommendedAdapterID == ""
	return report, nil
}

// RouteProbeHost reduces what a user would paste (URL, host:port, bare domain)
// to the host that has to be resolved.
func RouteProbeHost(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if at := strings.LastIndex(value, "@"); at >= 0 && strings.Contains(value[:at], "//") {
		value = value[at+1:]
	}
	value = routeProbeScheme.ReplaceAllString(value, "")
	if cut := strings.IndexAny(value, "/?#"); cut >= 0 {
		value = value[:cut]
	}
	if ip := net.ParseIP(value); ip != nil {
		return ip.String()
	}
	// Drop a port suffix, but keep IPv6 literals intact.
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	return strings.TrimSuffix(strings.ToLower(value), ".")
}

// RouteProbePort returns the port the user typed in the rule value, or 0 when
// none was given. An internal app on :8081 has to be probed on 8081, otherwise
// every adapter looks dead.
func RouteProbePort(raw string) int {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0
	}
	value = routeProbeScheme.ReplaceAllString(value, "")
	if cut := strings.IndexAny(value, "/?#"); cut >= 0 {
		value = value[:cut]
	}
	if net.ParseIP(value) != nil {
		return 0
	}
	_, portText, err := net.SplitHostPort(value)
	if err != nil {
		return 0
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 || port > 65535 {
		return 0
	}
	return port
}

func routeProbePortsFor(raw string) []int {
	ports := make([]int, 0, len(routeProbePorts)+1)
	if port := RouteProbePort(raw); port > 0 {
		ports = append(ports, port)
	}
	for _, port := range routeProbePorts {
		duplicate := false
		for _, existing := range ports {
			if existing == port {
				duplicate = true
				break
			}
		}
		if !duplicate {
			ports = append(ports, port)
		}
	}
	return ports
}

func routeProbeAddresses(host string) []string {	if ip := net.ParseIP(host); ip != nil {
		if v4 := ip.To4(); v4 != nil {
			return []string{v4.String()}
		}
		return []string{ip.String()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), routeProbeResolveTimeout)
	defer cancel()
	resolved, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
	if err != nil {
		return nil
	}
	var addresses []string
	seen := map[string]struct{}{}
	for _, ip := range resolved {
		v4 := ip.To4()
		if v4 == nil {
			continue
		}
		text := v4.String()
		if _, duplicate := seen[text]; duplicate {
			continue
		}
		seen[text] = struct{}{}
		addresses = append(addresses, text)
		if len(addresses) >= routeProbeMaxAddresses {
			break
		}
	}
	return addresses
}

// Virtual adapters (Hyper-V, VMware, tunnels) are never real internet paths, so
// probing them would only add noise and delay.
func routeProbeCandidates(adapters []AdapterView) []AdapterView {
	var candidates []AdapterView
	for _, adapter := range adapters {
		if adapter.IsVirtual || !adapter.Operational || adapter.Address == "" {
			continue
		}
		if adapter.Kind != "ethernet" && adapter.Kind != "wifi" {
			continue
		}
		candidates = append(candidates, adapter)
	}
	return candidates
}

func probeAdapterTarget(adapter AdapterView, addresses []string, ports []int) RouteProbeHop {
	hop := RouteProbeHop{
		AdapterID:     adapter.ID,
		AdapterName:   adapter.Name,
		Kind:          adapter.Kind,
		SourceAddress: adapter.Address,
	}
	var details []string
	for _, address := range addresses {
		for _, port := range ports {
			latency, err := probeAdapterConnect(adapter, address, port, routeProbeDialTimeout)
			target := net.JoinHostPort(address, fmt.Sprintf("%d", port))
			if err == nil {
				hop.OK = true
				hop.Target = target
				hop.LatencyMS = latency.Milliseconds()
				if len(details) > 0 {
					hop.Detail = strings.Join(details, "；")
				}
				return hop
			}
			details = append(details, fmt.Sprintf("%s 不通", target))
		}
	}
	hop.Detail = strings.Join(details, "；")
	return hop
}
