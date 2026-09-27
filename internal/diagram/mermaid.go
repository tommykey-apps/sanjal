// Package diagram は hynt の Report から Mermaid の図 (graph LR) を作る。
// 図のルールは docs/SPEC.md の「図のルール」
package diagram

import (
	"fmt"
	"strings"

	"github.com/tommykey-apps/hynt"
	"github.com/tommykey-apps/hynt/link"
	"github.com/tommykey-apps/hynt/route"
)

// Mermaid は host を中心にした星形の図を返す。Report が同じなら出力も同じ
// (hynt が名前でソートして返すので、ここでは並べ替えない)
func Mermaid(r hynt.Report) string {
	var b strings.Builder
	b.WriteString("graph LR\n")
	fmt.Fprintf(&b, "  host{{%s}}\n", quote(r.Host))

	routesByDev := map[string][]route.Route{}
	for _, rt := range r.Routes {
		routesByDev[rt.Dev] = append(routesByDev[rt.Dev], rt)
	}

	drawn := map[string]bool{}
	for _, l := range r.Links {
		if !drawable(l, routesByDev) {
			continue
		}
		drawn[l.Name] = true
		id := nodeID("l", l.Name)
		fmt.Fprintf(&b, "  %s[%s]\n", id, quote(linkLabel(l)))
		if l.Kind == link.VPN {
			fmt.Fprintf(&b, "  host -->|%s| %s\n", quote(l.Impl), id)
		} else {
			fmt.Fprintf(&b, "  host --- %s\n", id)
		}
		for _, rt := range routesByDev[l.Name] {
			dst := rt.Dst
			if rt.Gateway != "" {
				dst += " via " + rt.Gateway
			}
			rid := nodeID("r", l.Name+"_"+dst)
			fmt.Fprintf(&b, "  %s([%s])\n", rid, quote(dst))
			if rt.Table != "main" {
				fmt.Fprintf(&b, "  %s -->|%s| %s\n", id, quote("table "+rt.Table), rid)
			} else {
				fmt.Fprintf(&b, "  %s --> %s\n", id, rid)
			}
		}
	}

	return b.String()
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
