// Package inbound は、外からどの口のどのポートに入ってこられるかを決める。
// 待ち受けソケットを口に割り当て、ファイアウォールの規則をたどって通すか塞ぐかを判定する。
// 判定の規則は docs/SPEC.md の「内向きの判定」
package inbound

import (
	"cmp"
	"net/netip"
	"slices"
	"strings"

	"github.com/tommykey-apps/hynt"
	"github.com/tommykey-apps/hynt/firewall"
	"github.com/tommykey-apps/hynt/link"
	"github.com/tommykey-apps/hynt/listen"
)

type Verdict int

const (
	Unchecked Verdict = iota // ファイアウォールを読めていない、またはマシンの中だけの口
	Allowed
	Blocked
	Partial // 送信元などの読めない条件があり、一部だけ通す (または一部だけ塞ぐ)
)

type Entry struct {
	Proto   string
	Port    int
	Process string
	Verdict Verdict
}

// Local は、マシンの中 (127.0.0.1 や ::1) でだけ待ち受けている口を入れる Of の鍵
const Local = ""

// Of は口の名前ごとに、入ってこられるポートを返す。マシンの中だけの口は鍵 Local に入る。
// 止まっている口 (DOWN) とアドレスの無い口には割り当てない
func Of(r hynt.Report) map[string][]Entry {
	checked := r.FirewallState == hynt.FirewallRead
	out := map[string][]Entry{}
	for _, s := range r.Listens {
		if isLocal(s) {
			out[Local] = add(out[Local], Entry{Proto: s.Proto, Port: s.Port, Process: s.Process})
			continue
		}
		for _, l := range r.Links {
			fams := families(s, l)
			if len(fams) == 0 {
				continue
			}
			e := Entry{Proto: s.Proto, Port: s.Port, Process: s.Process}
			if checked {
				e.Verdict = judgeFamilies(r.Firewall, l.Name, s.Proto, s.Port, fams)
			}
			out[l.Name] = add(out[l.Name], e)
		}
	}
	for k := range out {
		slices.SortFunc(out[k], func(a, b Entry) int {
			return cmp.Or(cmp.Compare(a.Port, b.Port), strings.Compare(a.Proto, b.Proto))
		})
	}
	return out
}

// IPv4 と IPv6 の両方で待ち受けると同じポートが 2 回来る。1 つにまとめ、判定が割れたら条件付き
func add(es []Entry, e Entry) []Entry {
	for i := range es {
		if es[i].Proto == e.Proto && es[i].Port == e.Port {
			if es[i].Process == "" {
				es[i].Process = e.Process
			}
			if es[i].Verdict != e.Verdict {
				es[i].Verdict = Partial
			}
			return es
		}
	}
	return append(es, e)
}

func isLocal(s listen.Socket) bool {
	if s.Dev == "lo" {
		return true
	}
	a, err := netip.ParseAddr(s.Addr)
	return err == nil && a.IsLoopback()
}

// families は、ソケット s に口 l から届く IP の系統 (4 / 6) を返す。届かなければ空
func families(s listen.Socket, l link.Link) []int {
	if l.State == "DOWN" || (s.Dev != "" && s.Dev != l.Name) {
		return nil
	}
	has := map[int]bool{}
	for _, a := range l.Addrs {
		p, err := netip.ParsePrefix(a)
		if err != nil {
			continue
		}
		switch s.Addr {
		case "0.0.0.0":
			has[4] = has[4] || p.Addr().Is4()
		case "::", "*":
			// Linux の既定 (bindv6only=0) では :: の待ち受けは IPv4 も受ける
			has[family(p.Addr())] = true
		default:
			if p.Addr().String() == s.Addr {
				has[family(p.Addr())] = true
			}
		}
	}
	var fams []int
	for _, f := range []int{4, 6} {
		if has[f] {
			fams = append(fams, f)
		}
	}
	return fams
}

func family(a netip.Addr) int {
	if a.Is4() {
		return 4
	}
	return 6
}

func judgeFamilies(chains []firewall.Chain, iif, proto string, port int, fams []int) Verdict {
	v := judge(chains, iif, proto, port, fams[0])
	for _, f := range fams[1:] {
		if judge(chains, iif, proto, port, f) != v {
			return Partial
		}
	}
	return v
}

// judge は、口 iif に外から来た新しい接続 (proto, port) を、基本のチェーンを優先度の順に通して判定する。
// どれか 1 つのチェーンが塞げば塞がれる
func judge(chains []firewall.Chain, iif, proto string, port int, fam int) Verdict {
	w := walker{chains: chains, iif: iif, proto: proto, port: port}
	blocked := false
	for _, c := range chains {
		if c.Hook != "input" || !familyApplies(c.Family, fam) {
			continue
		}
		res := w.walk(c, 0)
		if res == pass {
			res = accept
			if c.Policy == "drop" {
				res = drop
			}
		}
		if res == drop {
			blocked = true
			break
		}
	}
	switch {
	case blocked && w.mayAccept, !blocked && w.mayDrop:
		return Partial
	case blocked:
		return Blocked
	}
	return Allowed
}

func familyApplies(chainFamily string, fam int) bool {
	switch chainFamily {
	case "ip":
		return fam == 4
	case "ip6":
		return fam == 6
	}
	return true
}

type outcome int

const (
	pass outcome = iota // 結果が決まらずチェーンの終わりまで来た、または return
	accept
	drop
)

// maxDepth は jump の入れ子の上限。規則が輪になっていても止まるように
const maxDepth = 16

type walker struct {
	chains []firewall.Chain
	iif    string
	proto  string
	port   int
	// 読めない条件の規則が「当たるかもしれない」ときに立てる。最後の判定を条件付きにする
	mayAccept, mayDrop bool
}

func (w *walker) walk(c firewall.Chain, depth int) outcome {
	if depth > maxDepth {
		w.mayAccept, w.mayDrop = true, true
		return pass
	}
	for _, r := range c.Rules {
		m := w.match(r)
		if m == no {
			continue
		}
		switch r.Verdict {
		case "accept":
			if m == yes {
				return accept
			}
			w.mayAccept = true
		case "drop", "reject":
			if m == yes {
				return drop
			}
			w.mayDrop = true
		case "jump", "goto":
			t, ok := w.chain(c, r.Target)
			if !ok {
				w.mayAccept, w.mayDrop = true, true
				continue
			}
			res := w.walk(t, depth+1)
			if m == maybe {
				w.mark(res)
				continue
			}
			if res != pass {
				return res
			}
			if r.Verdict == "goto" {
				// goto 先から戻ると、呼び出したチェーンも終わる
				return pass
			}
		case "return":
			if m == yes {
				return pass
			}
			w.mayAccept, w.mayDrop = true, true
		case "unknown":
			w.mayAccept, w.mayDrop = true, true
		}
	}
	return pass
}

func (w *walker) mark(res outcome) {
	switch res {
	case accept:
		w.mayAccept = true
	case drop:
		w.mayDrop = true
	}
}

// jump 先は同じ表の中にある
func (w *walker) chain(from firewall.Chain, name string) (firewall.Chain, bool) {
	for _, c := range w.chains {
		if c.Family == from.Family && c.Table == from.Table && c.Name == name {
			return c, true
		}
	}
	return firewall.Chain{}, false
}

type matched int

const (
	no matched = iota
	yes
	maybe
)

func (w *walker) match(r firewall.Rule) matched {
	// 外から繋ぎに来るのは新しい接続。established などだけの規則には当たらない
	if len(r.CtStates) > 0 && !slices.Contains(r.CtStates, "new") {
		return no
	}
	if len(r.Iifnames) > 0 && !slices.ContainsFunc(r.Iifnames, func(n string) bool { return ifnameMatch(n, w.iif) }) {
		return no
	}
	if len(r.Protos) > 0 && !slices.Contains(r.Protos, w.proto) {
		return no
	}
	if len(r.Dports) > 0 && !slices.ContainsFunc(r.Dports, func(p firewall.PortRange) bool { return p.From <= w.port && w.port <= p.To }) {
		return no
	}
	if len(r.Unknown) > 0 {
		return maybe
	}
	return yes
}

// nft の iifname は末尾の * で前方一致になる ("eth*")
func ifnameMatch(pattern, name string) bool {
	if p, ok := strings.CutSuffix(pattern, "*"); ok {
		return strings.HasPrefix(name, p)
	}
	return pattern == name
}
