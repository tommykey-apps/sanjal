package inbound

import (
	"reflect"
	"testing"

	"github.com/tommykey-apps/hynt"
	"github.com/tommykey-apps/hynt/firewall"
	"github.com/tommykey-apps/hynt/link"
	"github.com/tommykey-apps/hynt/listen"
)

// 架空の機器。アドレスは文書用 (192.0.2.0/24、198.51.100.0/24、2001:db8::/32)
var links = []link.Link{
	{Name: "eth0", Kind: link.Ethernet, State: "UP", Addrs: []string{"192.0.2.10/24", "2001:db8::10/64"}},
	{Name: "eth1", Kind: link.Ethernet, State: "DOWN", Addrs: []string{"198.51.100.9/24"}},
	{Name: "tun0", Kind: link.VPN, State: "UNKNOWN", Addrs: []string{"100.64.0.1/32"}},
	{Name: "veth1", Kind: link.Virtual, State: "UP"},
}

func TestOfAssign(t *testing.T) {
	r := hynt.Report{
		Links: links,
		Listens: []listen.Socket{
			{Proto: "tcp", Addr: "0.0.0.0", Port: 22, Process: "sshd"},
			{Proto: "tcp", Addr: "::", Port: 22},                    // 上と同じポート。1 つにまとまる
			{Proto: "tcp", Addr: "127.0.0.1", Port: 631},            // マシンの中だけ
			{Proto: "udp", Addr: "127.0.0.53", Port: 53, Dev: "lo"}, // マシンの中だけ
			{Proto: "tcp", Addr: "100.64.0.1", Port: 8080},          // tun0 のアドレスだけ
			{Proto: "udp", Addr: "*", Port: 5353, Dev: "eth0"},      // eth0 に縛っている
			{Proto: "tcp", Addr: "203.0.113.1", Port: 9999},         // どの口のアドレスでもない
		},
		FirewallState: hynt.FirewallDenied,
	}
	want := map[string][]Entry{
		Local:  {{Proto: "udp", Port: 53}, {Proto: "tcp", Port: 631}},
		"eth0": {{Proto: "tcp", Port: 22, Process: "sshd"}, {Proto: "udp", Port: 5353}},
		"tun0": {{Proto: "tcp", Port: 22, Process: "sshd"}, {Proto: "tcp", Port: 8080}},
		// eth1 は DOWN、veth1 はアドレスが無いので割り当てない
	}
	if got := Of(r); !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}

// Arch の既定 (/etc/nftables.conf) と同じ形に、口ごとの許可と jump 先を足した規則
var chains = []firewall.Chain{
	{Family: "inet", Table: "filter", Name: "input", Hook: "input", Policy: "drop", Rules: []firewall.Rule{
		{CtStates: []string{"invalid"}, Verdict: "drop"},
		{CtStates: []string{"established", "related"}, Verdict: "accept"},
		{Iifnames: []string{"lo"}, Verdict: "accept"},
		{Protos: []string{"icmp", "ipv6-icmp"}, Verdict: "accept"},
		{Protos: []string{"tcp"}, Dports: ports(22, 22), Verdict: "accept"},
		{Iifnames: []string{"tun*"}, Protos: []string{"tcp"}, Dports: ports(8000, 8100), Verdict: "accept"},
		{Verdict: "jump", Target: "extra"},
		{Verdict: "reject"},
	}},
	{Family: "inet", Table: "filter", Name: "extra", Rules: []firewall.Rule{
		{Unknown: []string{"ip saddr == 192.0.2.0/24"}, Protos: []string{"tcp"}, Dports: ports(9000, 9000), Verdict: "accept"},
		{Protos: []string{"udp"}, Dports: ports(5353, 5353), Verdict: "accept"},
		{Protos: []string{"tcp"}, Dports: ports(7000, 7000), Verdict: "return"},
	}},
	{Family: "ip6", Table: "v6", Name: "input", Hook: "input", Prio: 10, Policy: "accept", Rules: []firewall.Rule{
		{Protos: []string{"udp"}, Dports: ports(5353, 5353), Verdict: "drop"},
	}},
}

func TestJudge(t *testing.T) {
	cases := []struct {
		name  string
		iif   string
		proto string
		port  int
		fam   int
		want  Verdict
	}{
		{"tcp 22 は全口で許可", "eth0", "tcp", 22, 4, Allowed},
		{"規則に無いポートは reject", "eth0", "tcp", 80, 4, Blocked},
		{"tun* の前方一致で許可", "tun0", "tcp", 8080, 4, Allowed},
		{"tun* に当たらない口は塞ぐ", "eth0", "tcp", 8080, 4, Blocked},
		{"送信元が読めない許可は条件付き", "eth0", "tcp", 9000, 4, Partial},
		{"jump 先で許可", "eth0", "udp", 5353, 4, Allowed},
		{"IPv6 だけ別の表で塞ぐ", "eth0", "udp", 5353, 6, Blocked},
		{"return で戻り、次の reject に当たる", "eth0", "tcp", 7000, 4, Blocked},
		{"established だけの規則には当たらない", "eth0", "udp", 1234, 4, Blocked},
	}
	for _, c := range cases {
		if got := judge(chains, c.iif, c.proto, c.port, c.fam); got != c.want {
			t.Errorf("%s: got %d want %d", c.name, got, c.want)
		}
	}
}

func TestJudgeNoChains(t *testing.T) {
	// 読めて 0 件なら、塞ぐものは無い
	if got := judge(nil, "eth0", "tcp", 80, 4); got != Allowed {
		t.Errorf("got %d", got)
	}
}

func TestJudgeLoop(t *testing.T) {
	loop := []firewall.Chain{
		{Family: "inet", Table: "t", Name: "in", Hook: "input", Policy: "accept", Rules: []firewall.Rule{{Verdict: "jump", Target: "a"}}},
		{Family: "inet", Table: "t", Name: "a", Rules: []firewall.Rule{{Verdict: "jump", Target: "a"}}},
	}
	if got := judge(loop, "eth0", "tcp", 80, 4); got != Partial {
		t.Errorf("輪になった jump で判定を決めつけている: %d", got)
	}
}

func TestOfFirewall(t *testing.T) {
	r := hynt.Report{
		Links: links,
		Listens: []listen.Socket{
			{Proto: "udp", Addr: "::", Port: 5353}, // eth0 では IPv4 は通り IPv6 は塞がれる
			{Proto: "tcp", Addr: "0.0.0.0", Port: 80},
		},
		Firewall:      chains,
		FirewallState: hynt.FirewallRead,
	}
	got := Of(r)["eth0"]
	want := []Entry{{Proto: "tcp", Port: 80, Verdict: Blocked}, {Proto: "udp", Port: 5353, Verdict: Partial}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got  %+v\nwant %+v", got, want)
	}
}

func ports(from, to int) []firewall.PortRange {
	return []firewall.PortRange{{From: from, To: to}}
}
