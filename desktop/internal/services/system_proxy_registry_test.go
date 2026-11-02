//go:build windows

package services

import (
	"fmt"
	"os"
	"testing"

	"golang.org/x/sys/windows/registry"
)

// This test drives the real Windows registry, because the bug it guards
// cannot be reproduced with the fake key: the fake key has no WinINET cache
// behind it, and that cache is exactly what used to undo the restore.
//
// It is opt-in. Set HIMIX_PROXY_REGISTRY_TEST=1 to run it; the machine's
// current proxy settings are saved and put back before the test returns.
func TestSystemProxyRestoreAgainstTheRealRegistry(t *testing.T) {
	if os.Getenv("HIMIX_PROXY_REGISTRY_TEST") != "1" {
		t.Skip("跳过真实注册表测试：设置 HIMIX_PROXY_REGISTRY_TEST=1 后运行（会临时改写代理设置并自动还原）")
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	defer key.Close()

	type val struct {
		exists bool
		num    uint64
		str    string
	}
	saved := map[string]val{}
	for _, n := range []string{"ProxyEnable", "ProxyServer", "ProxyOverride"} {
		if num, _, e := key.GetIntegerValue(n); e == nil {
			saved[n] = val{true, num, ""}
		} else if s, _, e := key.GetStringValue(n); e == nil {
			saved[n] = val{true, 0, s}
		} else {
			saved[n] = val{}
		}
	}
	defer func() {
		for _, n := range []string{"ProxyEnable", "ProxyServer", "ProxyOverride"} {
			v := saved[n]
			switch {
			case !v.exists:
				key.DeleteValue(n)
			case n == "ProxyEnable":
				key.SetDWordValue(n, uint32(v.num))
			default:
				key.SetStringValue(n, v.str)
			}
		}
		notifyProxyChanged()
	}()

	show := func(label string) {
		e, _, _ := key.GetIntegerValue("ProxyEnable")
		s, _, se := key.GetStringValue("ProxyServer")
		o, _, oe := key.GetStringValue("ProxyOverride")
		fmt.Printf("  [%s] ProxyEnable=%d ProxyServer存在=%v(%q) ProxyOverride存在=%v(%.30q)\n",
			label, e, se == nil, s, oe == nil, o)
	}

	probe := func() (uint64, bool, bool) {
		e, _, _ := key.GetIntegerValue("ProxyEnable")
		_, _, se := key.GetStringValue("ProxyServer")
		_, _, oe := key.GetStringValue("ProxyOverride")
		return e, se == nil, oe == nil
	}

	run := func(title string, setup func(), wantEnable uint64, wantServer string, wantOverride string) {
		fmt.Printf("\n=== %s ===\n", title)
		setup()
		show("原始")
		if err := enableSystemProxy(10801, 10800); err != nil {
			t.Fatalf("enableSystemProxy: %v", err)
		}
		show("启用后")
		msg, err := restoreSystemProxyDetailed()
		if err != nil {
			t.Fatalf("restoreSystemProxy: %v", err)
		}
		fmt.Printf("  [恢复返回] %q\n", msg)
		show("停止后")
		e, _, _ := probe()
		gotServer, _, _ := key.GetStringValue("ProxyServer")
		gotOverride, _, _ := key.GetStringValue("ProxyOverride")
		if e != wantEnable || gotServer != wantServer || gotOverride != wantOverride {
			t.Errorf("%s 还原不符：期望 ProxyEnable=%d ProxyServer=%q ProxyOverride=%q，实际 ProxyEnable=%d ProxyServer=%q ProxyOverride=%q",
				title, wantEnable, wantServer, wantOverride, e, gotServer, gotOverride)
		} else {
			fmt.Printf("  ✅ %s 还原正确\n", title)
		}
	}

	run("场景 A：原机无代理设置", func() {
		key.DeleteValue("ProxyEnable")
		key.DeleteValue("ProxyServer")
		key.DeleteValue("ProxyOverride")
	}, 0, "", "")

	run("场景 B：原机残留上一轮的 Himix 值", func() {
		key.DeleteValue("ProxyEnable")
		key.SetStringValue("ProxyServer", "http=127.0.0.1:10801;https=127.0.0.1:10801;socks=127.0.0.1:10800")
		key.SetStringValue("ProxyOverride", systemProxyBypass)
	}, 0, "", "")

	run("场景 C：用户本来就有自己的代理", func() {
		key.SetDWordValue("ProxyEnable", 1)
		key.SetStringValue("ProxyServer", "http=192.168.1.9:8888")
		key.SetStringValue("ProxyOverride", "corp.local;<local>")
	}, 1, "http=192.168.1.9:8888", "corp.local;<local>")
}
