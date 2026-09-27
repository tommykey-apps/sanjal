package diagram

import (
	"testing"

	"github.com/tommykey-apps/hynt"
	"github.com/tommykey-apps/hynt/link"
	"github.com/tommykey-apps/hynt/neigh"
	"github.com/tommykey-apps/hynt/route"
	"github.com/tommykey-apps/hynt/xfrm"
)

// この機器の hynt.Collect の結果を切り詰めたもの
var sample = hynt.Report{
	Schema: 1,
	Host:   "arch",
	Links: []link.Link{
		{Name: "eno1", Kind: link.Ethernet, State: "DOWN"},
		{Name: "tailscale0", Kind: link.VPN, Impl: "tun", State: "UNKNOWN", Addrs: []string{"100.64.0.1/32"}},
		{Name: "veth1", Kind: link.Virtual, Impl: "veth", State: "UP"},
		{Name: "wlp2s0", Kind: link.Wifi, State: "UP", Addrs: []string{"192.0.2.132/24"}},
	},
	Routes: []route.Route{
		{Dst: "100.64.0.2", Dev: "tailscale0", Table: "52"},
		{Dst: "default", Gateway: "192.0.2.1", Dev: "wlp2s0", Table: "main"},
		{Dst: "192.0.2.0/24", Dev: "wlp2s0", Table: "main"},
	},
	Neighs: []neigh.Neigh{
		{Dst: "172.17.0.2", Lladdr: "02:42:ac:11:00:02", Dev: "docker0", State: "STALE"}, // docker0 は Links に無いので描かない
		{Dst: "192.0.2.1", Lladdr: "02:00:00:00:00:01", Dev: "wlp2s0", State: "REACHABLE"},
		{Dst: "192.0.2.20", Lladdr: "11:22:33:44:55:66", Dev: "wlp2s0", State: "STALE"},
		{Dst: "2001:db8:1::1", Lladdr: "02:00:00:00:00:01", Dev: "wlp2s0", State: "REACHABLE"}, // 192.0.2.1 と同じ機器
	},
	Policies: []xfrm.Policy{
		{Src: "10.0.0.5/32", Dst: "10.233.0.0/18", Gateway: "203.0.113.9", Mode: "tunnel"},
	},
}

func TestMermaid(t *testing.T) {
	want := `graph LR
  host{{"arch"}}
  l_eno1["eno1<br/>ethernet (DOWN)"]
  host --- l_eno1
  l_tailscale0["tailscale0<br/>vpn tun<br/>100.64.0.1/32"]
  host -->|"tun"| l_tailscale0
  r_tailscale0_100_64_0_2(["100.64.0.2"])
  l_tailscale0 -->|"table 52"| r_tailscale0_100_64_0_2
  l_wlp2s0["wlp2s0<br/>wifi<br/>192.0.2.132/24"]
  host --- l_wlp2s0
  r_wlp2s0_default_via_192_0_2_1(["default via 192.0.2.1"])
  l_wlp2s0 --> r_wlp2s0_default_via_192_0_2_1
  r_wlp2s0_192_0_2_0_24(["192.0.2.0/24"])
  l_wlp2s0 --> r_wlp2s0_192_0_2_0_24
  n_wlp2s0_02_00_00_00_00_01("192.0.2.1<br/>2001:db8:1::1<br/>02:00:00:00:00:01")
  l_wlp2s0 -.- n_wlp2s0_02_00_00_00_00_01
  n_wlp2s0_11_22_33_44_55_66("192.0.2.20<br/>11:22:33:44:55:66")
  l_wlp2s0 -.- n_wlp2s0_11_22_33_44_55_66
  p_10_233_0_0_18(["10.233.0.0/18"])
  host -->|"ipsec via 203.0.113.9"| p_10_233_0_0_18
`
	if got := Mermaid(sample); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestQuote(t *testing.T) {
	if got := quote(`a"b`); got != `"a#quot;b"` {
		t.Errorf("got %s", got)
	}
}
