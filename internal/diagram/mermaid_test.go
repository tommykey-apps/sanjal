package diagram

import (
	"reflect"
	"strings"
	"testing"

	"github.com/tommykey-apps/hynt"
	"github.com/tommykey-apps/hynt/link"
	"github.com/tommykey-apps/hynt/neigh"
	"github.com/tommykey-apps/hynt/route"
	"github.com/tommykey-apps/hynt/xfrm"
)

// 架空の機器。アドレスは文書用 (192.0.2.0/24、198.51.100.0/24、203.0.113.0/24、2001:db8::/32)、
// MAC は文書用 (00:00:5e:00:53:xx)。開発機の値は使わない
var sample = hynt.Report{
	Schema: 1,
	Host:   "box",
	Links: []link.Link{
		{Name: "br0", Kind: link.Bridge, Impl: "bridge", State: "UP", Addrs: []string{"198.51.100.1/24"}},
		{Name: "eth0", Kind: link.Ethernet, State: "DOWN"},
		{Name: "tun0", Kind: link.VPN, Impl: "tun", State: "UNKNOWN", Addrs: []string{"100.64.0.1/32"}},
		{Name: "veth1", Kind: link.Virtual, Impl: "veth", State: "UP", Master: "br0"},
		{Name: "wlan0", Kind: link.Wifi, State: "UP", Addrs: []string{"192.0.2.10/24"}},
	},
	Routes: []route.Route{
		{Dst: "198.51.100.0/24", Dev: "br0", Table: "main"},
		{Dst: "100.64.0.2", Dev: "tun0", Table: "100"},
		{Dst: "default", Gateway: "192.0.2.1", Dev: "wlan0", Table: "main"},
		{Dst: "default", Gateway: "fe80::200:5eff:fe00:5301", Dev: "wlan0", Table: "main"}, // 192.0.2.1 と同じ機器 (EUI-64)
		{Dst: "192.0.2.0/24", Dev: "wlan0", Table: "main"},
	},
	Rules: []route.Rule{
		{Priority: 0, Selector: "from all", Table: "local"},
		{Priority: 100, Selector: "from all", Table: "100"}, // default を持たないので main へ進む
		{Priority: 32766, Selector: "from all", Table: "main"},
	},
	Neighs: []neigh.Neigh{
		{Dst: "198.51.100.2", Lladdr: "00:00:5e:00:53:02", Dev: "br0", State: "STALE"},
		{Dst: "203.0.113.5", Lladdr: "00:00:5e:00:53:09", Dev: "nowhere0", State: "STALE"}, // Links に無いので描かない
		{Dst: "192.0.2.1", Lladdr: "00:00:5e:00:53:01", Dev: "wlan0", State: "REACHABLE"},
		{Dst: "192.0.2.20", Lladdr: "00:00:5e:00:53:03", Dev: "wlan0", State: "STALE"},
		{Dst: "2001:db8:1::1", Lladdr: "00:00:5e:00:53:01", Dev: "wlan0", State: "REACHABLE"}, // 192.0.2.1 と同じ機器
	},
	Policies: []xfrm.Policy{
		{Src: "192.0.2.10/32", Dst: "203.0.113.0/24", Gateway: "203.0.113.1", Mode: "tunnel"},
	},
}

func TestMermaid(t *testing.T) {
	want := `graph LR
  host{{"box"}}
  subgraph z_inner["マシンの中 (外へ出ない)"]
    l_br0["br0<br/>bridge<br/>198.51.100.1/24"]
    r_br0_main_198_51_100_0_24(["198.51.100.0/24"])
    n_br0_00_00_5e_00_53_02("198.51.100.2<br/>00:00:5e:00:53:02")
  end
  subgraph z_vpn["VPN"]
    l_tun0["tun0<br/>vpn tun<br/>100.64.0.1/32"]
    r_tun0_100_100_64_0_2(["100.64.0.2"])
    p_203_0_113_0_24(["203.0.113.0/24"])
  end
  subgraph z_lan["LAN"]
    l_eth0["eth0<br/>ethernet (DOWN)"]
    l_wlan0["wlan0<br/>wifi<br/>192.0.2.10/24"]
    r_wlan0_main_192_0_2_0_24(["192.0.2.0/24"])
    n_wlan0_00_00_5e_00_53_01("ゲートウェイ<br/>192.0.2.1<br/>2001:db8:1::1<br/>00:00:5e:00:53:01")
    n_wlan0_00_00_5e_00_53_03("192.0.2.20<br/>00:00:5e:00:53:03")
  end
  inet(("インターネット"))
  host --- l_br0
  l_br0 --> r_br0_main_198_51_100_0_24
  host --- l_eth0
  host -->|"tun"| l_tun0
  l_tun0 -->|"table 100"| r_tun0_100_100_64_0_2
  host --- l_wlan0
  l_wlan0 --> n_wlan0_00_00_5e_00_53_01
  n_wlan0_00_00_5e_00_53_01 --> inet
  l_wlan0 --> r_wlan0_main_192_0_2_0_24
  l_br0 -.- n_br0_00_00_5e_00_53_02
  l_wlan0 -.- n_wlan0_00_00_5e_00_53_03
  host -->|"ipsec via 203.0.113.1"| p_203_0_113_0_24
`
	if got := Mermaid(sample); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// VPN が既定経路を奪うと、main の default は使われず、VPN の口からインターネットへ線が出る
func TestMermaidFullTunnel(t *testing.T) {
	r := hynt.Report{
		Host: "box",
		Links: []link.Link{
			{Name: "wg0", Kind: link.VPN, Impl: "wireguard", State: "UNKNOWN", Addrs: []string{"198.51.100.2/32"}},
			{Name: "wlan0", Kind: link.Wifi, State: "UP", Addrs: []string{"192.0.2.10/24"}},
		},
		Routes: []route.Route{
			{Dst: "default", Dev: "wg0", Table: "200"},
			{Dst: "default", Gateway: "192.0.2.1", Dev: "wlan0", Table: "main"},
		},
		Rules: []route.Rule{
			{Priority: 10, Selector: "from all", Table: "main", SuppressPrefixlen: new(0)},
			{Priority: 20, Selector: "not from all fwmark 0x1", Table: "200"},
			{Priority: 32766, Selector: "from all", Table: "main"},
		},
	}
	got := Mermaid(r)
	for _, want := range []string{
		`l_wg0 --> inet`,
		`r_wlan0_main_default_via_192_0_2_1(["default via 192.0.2.1<br/>(使われていない)"])`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("%q が無い:\n%s", want, got)
		}
	}
	if strings.Contains(got, "g_wlan0") {
		t.Errorf("使われない default にゲートウェイを描いている:\n%s", got)
	}
}

// br0 に eth0 を繋いだ構成では、br0 は LAN の枠に入り、eth0 への線は host ではなく br0 から引く
func TestMermaidBridgedEthernet(t *testing.T) {
	r := hynt.Report{
		Host: "box",
		Links: []link.Link{
			{Name: "br0", Kind: link.Bridge, Impl: "bridge", State: "UP", Addrs: []string{"192.0.2.10/24"}},
			{Name: "eth0", Kind: link.Ethernet, State: "UP", Master: "br0"},
		},
	}
	got := Mermaid(r)
	for _, want := range []string{
		"subgraph z_lan[\"LAN\"]\n    l_br0[",
		"  l_br0 --- l_eth0\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("%q が無い:\n%s", want, got)
		}
	}
	if strings.Contains(got, "host --- l_eth0") {
		t.Errorf("ブリッジの下の口に host から線を引いている:\n%s", got)
	}
}

func TestClassify(t *testing.T) {
	gw := map[string][]route.Route{"vlan10": {{Dst: "default", Gateway: "192.0.2.1", Dev: "vlan10", Table: "main"}}}
	members := map[string][]link.Link{
		"br0":     {{Name: "eth0", Kind: link.Ethernet, Master: "br0"}},
		"docker0": {{Name: "veth1", Kind: link.Virtual, Master: "docker0"}},
	}
	cases := []struct {
		l    link.Link
		want zone
	}{
		{link.Link{Name: "wg0", Kind: link.VPN}, zoneVPN},
		{link.Link{Name: "wlan0", Kind: link.Wifi}, zoneLAN},
		{link.Link{Name: "eth0", Kind: link.Ethernet}, zoneLAN},
		{link.Link{Name: "br0", Kind: link.Bridge}, zoneLAN},       // 物理の口を持つブリッジ
		{link.Link{Name: "docker0", Kind: link.Bridge}, zoneInner}, // veth だけのブリッジ
		{link.Link{Name: "vlan10", Kind: link.Virtual}, zoneLAN},   // 既定経路のゲートウェイがある
		{link.Link{Name: "dummy0", Kind: link.Virtual}, zoneInner},
	}
	for _, c := range cases {
		if got := classify(c.l, members, gw); got != c.want {
			t.Errorf("%s: got %d want %d", c.l.Name, got, c.want)
		}
	}
}

func TestInternetRoutes(t *testing.T) {
	mainDefault := route.Route{Dst: "default", Gateway: "192.0.2.1", Dev: "wlan0", Table: "main"}
	vpnDefault := route.Route{Dst: "default", Dev: "wg0", Table: "200"}
	half1 := route.Route{Dst: "0.0.0.0/1", Dev: "tun0", Table: "main"}
	half2 := route.Route{Dst: "128.0.0.0/1", Dev: "tun0", Table: "main"}
	mainRule := route.Rule{Priority: 32766, Selector: "from all", Table: "main"}
	cases := []struct {
		name   string
		routes []route.Route
		rules  []route.Rule
		want   []route.Route
	}{
		{"rule が無ければ main", []route.Route{mainDefault}, nil, []route.Route{mainDefault}},
		{"先の規則のテーブルが勝つ", []route.Route{mainDefault, vpnDefault},
			[]route.Rule{{Priority: 10, Selector: "from all", Table: "200"}, mainRule}, []route.Route{vpnDefault}},
		{"fwmark 付きの規則は普通の通信に当たらない", []route.Route{mainDefault, vpnDefault},
			[]route.Rule{{Priority: 10, Selector: "from all fwmark 0x1", Table: "200"}, mainRule}, []route.Route{mainDefault}},
		{"unreachable の規則は飛ばす", []route.Route{mainDefault},
			[]route.Rule{{Priority: 10, Selector: "from all", Action: "unreachable"}, mainRule}, []route.Route{mainDefault}},
		{"suppress_prefixlength 0 で main の default を無視し、not fwmark のテーブルへ", []route.Route{mainDefault, vpnDefault},
			[]route.Rule{
				{Priority: 10, Selector: "from all", Table: "main", SuppressPrefixlen: new(0)},
				{Priority: 20, Selector: "not from all fwmark 0x1", Table: "200"},
				mainRule,
			}, []route.Route{vpnDefault}},
		{"/1 の 2 本は default より優先", []route.Route{half1, half2, mainDefault}, nil, []route.Route{half1, half2}},
		{"既定経路が無い", []route.Route{{Dst: "192.0.2.0/24", Dev: "wlan0", Table: "main"}}, nil, nil},
	}
	for _, c := range cases {
		got := internetRoutes(c.routes, c.rules)
		var want map[route.Route]bool
		if c.want != nil {
			want = map[route.Route]bool{}
			for _, rt := range c.want {
				want[rt] = true
			}
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %v want %v", c.name, got, want)
		}
	}
}

func TestEUI64MAC(t *testing.T) {
	cases := map[string]string{
		"fe80::200:5eff:fe00:5301": "00:00:5e:00:53:01",
		"fe80::1":                  "", // 手で付けたアドレス
		"192.0.2.1":                "",
		"":                         "",
	}
	for in, want := range cases {
		if got := eui64MAC(in); got != want {
			t.Errorf("%q: got %q want %q", in, got, want)
		}
	}
}

func TestQuote(t *testing.T) {
	if got := quote(`a"b`); got != `"a#quot;b"` {
		t.Errorf("got %s", got)
	}
}
