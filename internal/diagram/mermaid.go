// Package diagram は hynt の Report から Mermaid の図 (graph LR) を作る。
// 図のルールは docs/SPEC.md の「図のルール」
package diagram

import (
	"fmt"
	"net/netip"
	"slices"
	"strings"

	"github.com/tommykey-apps/hynt"
	"github.com/tommykey-apps/hynt/link"
	"github.com/tommykey-apps/hynt/neigh"
	"github.com/tommykey-apps/hynt/route"
)

// 区分
type zone int

const (
	zoneInner zone = iota // マシンの中 (docker のブリッジなど)
	zoneLAN               // 家や職場の LAN
	zoneVPN
)

var zoneTitles = []struct{ id, title string }{
	{"z_inner", "マシンの中 (外へ出ない)"},
	{"z_lan", "LAN"},
	{"z_vpn", "VPN"},
}

const internetID = "inet"

// Mermaid は host から「マシンの中 / LAN / VPN」の枠を経てインターネットへ至る図を返す。
// Report が同じなら出力も同じ (hynt が名前でソートして返すので、ここでは並べ替えない)
func Mermaid(r hynt.Report) string {
	routesByDev := map[string][]route.Route{}
	for _, rt := range r.Routes {
		routesByDev[rt.Dev] = append(routesByDev[rt.Dev], rt)
	}
	membersOf := map[string][]link.Link{}
	for _, l := range r.Links {
		if l.Master != "" {
			membersOf[l.Master] = append(membersOf[l.Master], l)
		}
	}
	toInternet := internetRoutes(r.Routes, r.Rules)

	// Why not: ノードと線を 1 本の文字列に順に書かない。Mermaid は subgraph の外で先に
	// 名前が出たノードを枠の外に置くので、ノードを枠ごとに宣言し終えてから線を書く
	nodes := make([][]string, len(zoneTitles))
	var edges []string
	node := func(z zone, format string, a ...any) { nodes[z] = append(nodes[z], fmt.Sprintf(format, a...)) }
	// OpenVPN の 0.0.0.0/1 と 128.0.0.0/1 は同じ線になるので、同じ線は 1 本にする
	edge := func(format string, a ...any) {
		if e := fmt.Sprintf(format, a...); !slices.Contains(edges, e) {
			edges = append(edges, e)
		}
	}

	drawn := map[string]bool{}
	zoneOf := map[string]zone{}
	for _, l := range r.Links {
		if drawable(l, routesByDev) {
			drawn[l.Name] = true
		}
	}

	// ゲートウェイは同じ LAN の機器 (ip neigh) にも出るので、その機器のノードを兼ねる
	gateways := map[string]string{} // dev + " " + gateway → ノード ID
	groups := groupNeighs(r.Neighs)

	usedInternet := false
	for _, l := range r.Links {
		if !drawn[l.Name] {
			continue
		}
		z := classify(l, membersOf, routesByDev)
		zoneOf[l.Name] = z
		id := nodeID("l", l.Name)
		node(z, "%s[%s]", id, quote(linkLabel(l)))
		switch {
		case l.Master != "" && drawn[l.Master]:
			// ブリッジや bond の下にいる物理の口は、host ではなく親から線を引く
			edge("%s --- %s", nodeID("l", l.Master), id)
		case l.Kind == link.VPN:
			edge("host -->|%s| %s", quote(l.Impl), id)
		default:
			edge("host --- %s", id)
		}

		for _, rt := range routesByDev[l.Name] {
			arrow := "-->"
			if rt.Table != "main" {
				arrow = "-->|" + quote("table "+rt.Table) + "|"
			}
			if !toInternet[rt] {
				dst := rt.Dst
				if rt.Gateway != "" {
					dst += " via " + rt.Gateway
				}
				label := dst
				if isDefault(rt.Dst) {
					label += "<br/>(使われていない)"
				}
				rid := nodeID("r", l.Name+"_"+rt.Table+"_"+dst)
				node(z, "%s([%s])", rid, quote(label))
				edge("%s %s %s", id, arrow, rid)
				continue
			}
			usedInternet = true
			if rt.Gateway == "" {
				// tun のように相手の決まった口は、口から直接インターネットへ出る。
				// Why not: この線に table 名を書かない。VPN の枠の見出しに重なって読めなくなった
				edge("%s --> %s", id, internetID)
				continue
			}
			key := l.Name + " " + rt.Gateway
			gid, seen := gateways[key]
			if !seen {
				for _, g := range groups {
					if g.dev == l.Name && (slices.Contains(g.ips, rt.Gateway) || eui64MAC(rt.Gateway) == g.mac) {
						gid = g.id() // 機器のノードは下で宣言する
						break
					}
				}
				if gid == "" {
					gid = nodeID("g", l.Name+"_"+rt.Gateway)
					node(z, "%s(%s)", gid, quote("ゲートウェイ<br/>"+rt.Gateway))
				}
				gateways[key] = gid
			}
			edge("%s %s %s", id, arrow, gid)
			if !seen {
				edge("%s --> %s", gid, internetID)
			}
		}
	}

	// 同じ LAN の機器 (ip neigh) はインタフェースの先に破線で置く。角丸の四角 (Mermaid の (...))。
	// 同じ機器は IPv4 と IPv6 で別の行として来るので、MAC ごとに 1 つのノードにまとめる
	isGateway := map[string]bool{}
	for _, gid := range gateways {
		isGateway[gid] = true
	}
	for _, g := range groups {
		if !drawn[g.dev] {
			continue
		}
		label := strings.Join(append(g.ips, g.mac), "<br/>")
		if isGateway[g.id()] {
			// 経路の矢印が既にあるので、破線は重ねない
			node(zoneOf[g.dev], "%s(%s)", g.id(), quote("ゲートウェイ<br/>"+label))
			continue
		}
		node(zoneOf[g.dev], "%s(%s)", g.id(), quote(label))
		edge("%s -.- %s", nodeID("l", g.dev), g.id())
	}

	// policy-based IPsec はインタフェースを持たないので host から直接引く
	for _, p := range r.Policies {
		pid := nodeID("p", p.Dst)
		node(zoneVPN, "%s([%s])", pid, quote(p.Dst))
		edge("host -->|%s| %s", quote("ipsec via "+p.Gateway), pid)
	}

	var b strings.Builder
	b.WriteString("graph LR\n")
	fmt.Fprintf(&b, "  host{{%s}}\n", quote(r.Host))
	// Why not: host をマシンの中の枠に入れない。線が枠をまたいで交差し、かえって読めなくなった。
	// Mermaid (dagre) は後に宣言した枠ほど上に置くので、本線の LAN が一番上に来るよう逆順に書く
	for _, z := range []zone{zoneInner, zoneVPN, zoneLAN} {
		ns := nodes[z]
		if len(ns) == 0 {
			continue
		}
		fmt.Fprintf(&b, "  subgraph %s[%s]\n", zoneTitles[z].id, quote(zoneTitles[z].title))
		for _, n := range ns {
			fmt.Fprintf(&b, "    %s\n", n)
		}
		b.WriteString("  end\n")
	}
	if usedInternet {
		fmt.Fprintf(&b, "  %s((%s))\n", internetID, quote("インターネット"))
	}
	for _, e := range edges {
		fmt.Fprintf(&b, "  %s\n", e)
	}
	return b.String()
}

// classify は口を区分に分ける。名前では決めず、hynt の種別と所属と経路で決める
func classify(l link.Link, membersOf map[string][]link.Link, routesByDev map[string][]route.Route) zone {
	switch l.Kind {
	case link.VPN:
		return zoneVPN
	case link.Ethernet, link.Wifi:
		return zoneLAN
	}
	// br0 に eth0 を繋いだ構成や bond は、物理の口を持つので LAN の一部
	for _, m := range membersOf[l.Name] {
		if m.Kind == link.Ethernet || m.Kind == link.Wifi {
			return zoneLAN
		}
	}
	// vlan のように物理の口が見えなくても、ゲートウェイ付きの既定経路があれば外へ繋がっている
	for _, rt := range routesByDev[l.Name] {
		if isDefault(rt.Dst) && rt.Gateway != "" {
			return zoneLAN
		}
	}
	return zoneInner
}

// isDefault は宛先が「どれにも当たらない宛先すべて」を表すか。
// OpenVPN は default を上書きせず 0.0.0.0/1 と 128.0.0.0/1 の 2 本で全経路を取る
func isDefault(dst string) bool {
	switch dst {
	case "default", "0.0.0.0/1", "128.0.0.0/1", "::/1", "8000::/1":
		return true
	}
	return false
}

func prefixLen(dst string) int {
	if dst == "default" {
		return 0
	}
	return 1
}

// internetRoutes は、fwmark の付かない普通の通信がインターネットへ出るときに使う経路を返す。
// ip rule を優先度の順にたどり、既定経路を持つ最初のテーブルで決まる。
// rule が無ければ (古い hynt や記録済みの Report) main だけを見る
func internetRoutes(routes []route.Route, rules []route.Rule) map[route.Route]bool {
	if len(rules) == 0 {
		rules = []route.Rule{{Selector: "from all", Table: "main"}}
	}
	for _, ru := range rules {
		if ru.Table == "" || ru.Action != "" || !appliesToAll(ru.Selector) {
			continue
		}
		var hit []route.Route
		for _, rt := range routes {
			if rt.Table != ru.Table || !isDefault(rt.Dst) {
				continue
			}
			// wg-quick は main に suppress_prefixlength 0 を付け、main の default を無視させる
			if ru.SuppressPrefixlen != nil && prefixLen(rt.Dst) <= *ru.SuppressPrefixlen {
				continue
			}
			hit = append(hit, rt)
		}
		if len(hit) == 0 {
			continue
		}
		// 0.0.0.0/1 と default が同じテーブルにあれば、長い方 (/1) だけが使われる
		longest := 0
		for _, rt := range hit {
			longest = max(longest, prefixLen(rt.Dst))
		}
		set := map[route.Route]bool{}
		for _, rt := range hit {
			if prefixLen(rt.Dst) == longest {
				set[rt] = true
			}
		}
		return set
	}
	return nil
}

// 普通の通信は fwmark を持たないので、fwmark 付きの規則には当たらず、not fwmark の規則には当たる
func appliesToAll(selector string) bool {
	return selector == "from all" ||
		(strings.HasPrefix(selector, "not from all fwmark ") && !strings.Contains(selector, " to "))
}

// eui64MAC は MAC から作られた IPv6 アドレス (EUI-64) から MAC を戻す。作られていなければ空。
// ルーターの IPv6 の既定経路は fe80:: を指し、hynt は fe80:: の neigh を捨てるので、
// IPv4 のゲートウェイと同じ機器かどうかは MAC で照合するしかない
func eui64MAC(ip string) string {
	a, err := netip.ParseAddr(ip)
	if err != nil || !a.Is6() || a.Is4In6() {
		return ""
	}
	b := a.As16()
	if b[11] != 0xff || b[12] != 0xfe {
		return ""
	}
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", b[8]^0x02, b[9], b[10], b[13], b[14], b[15])
}

type neighGroup struct {
	dev, mac string
	ips      []string
}

func (g neighGroup) id() string { return nodeID("n", g.dev+"_"+g.mac) }

// 最初に現れた順を保つ。hynt が dev と IP でソートして返すので、出力は決定的になる
func groupNeighs(ns []neigh.Neigh) []neighGroup {
	var groups []neighGroup
	index := map[string]int{}
	for _, n := range ns {
		key := n.Dev + " " + n.Lladdr
		i, ok := index[key]
		if !ok {
			i = len(groups)
			index[key] = i
			groups = append(groups, neighGroup{dev: n.Dev, mac: n.Lladdr})
		}
		groups[i].ips = append(groups[i].ips, n.Dst)
	}
	return groups
}

// veth のようにアドレスも経路も無い仮想インタフェースは描かない。図が veth だらけになる
func drawable(l link.Link, routesByDev map[string][]route.Route) bool {
	return l.Kind != link.Virtual || len(l.Addrs) > 0 || len(routesByDev[l.Name]) > 0
}

func linkLabel(l link.Link) string {
	s := l.Name + "<br/>" + string(l.Kind)
	if l.Impl != "" && l.Impl != string(l.Kind) {
		s += " " + l.Impl
	}
	// 止まっているものだけ状態を書く。UP と UNKNOWN (tun は常に UNKNOWN) は書かない
	if l.State == "DOWN" {
		s += " (DOWN)"
	}
	for _, a := range l.Addrs {
		s += "<br/>" + a
	}
	return s
}

// Why not: %q を使わない。Go の \" は Mermaid では引用符にならず、図が壊れる。
// Mermaid の実体参照 #quot; に置き換える。改行は <br/> で書くので \n は来ない前提
func quote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, "#quot;") + `"`
}

// Mermaid のノード ID に使えるのは英数字と _ だけ。名前の他の文字は _ に落とす
func nodeID(prefix, name string) string {
	var b strings.Builder
	b.WriteString(prefix)
	b.WriteByte('_')
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
