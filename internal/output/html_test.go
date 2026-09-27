package output

import (
	"strings"
	"testing"
	"time"

	"github.com/tommykey-apps/hynt"
	"github.com/tommykey-apps/hynt/link"
	"github.com/tommykey-apps/hynt/neigh"
)

var now = time.Date(2026, 9, 26, 12, 34, 0, 0, time.UTC)

// Why not: mermaid.js を <script> に直接埋めるので、中に </script があると HTML が途中で切れる。
// mermaid を更新したらここで気づけるようにする
func TestMermaidJSEmbeddable(t *testing.T) {
	b, err := assets.ReadFile("assets/mermaid.min.js")
	if err != nil {
		t.Fatal(err)
	}
	s := strings.ToLower(string(b))
	if strings.Contains(s, "</script") || strings.Contains(s, "<script") {
		t.Fatal("mermaid.min.js に <script または </script が含まれる。そのままは埋め込めない")
	}
}

func TestHTML(t *testing.T) {
	r := hynt.Report{
		Host:        `arch<x>`,
		Links:       []link.Link{{Name: "wlp2s0", Kind: link.Wifi, State: "UP", Addrs: []string{"192.0.2.132/24"}}},
		IPsecDenied: true,
		Neighs: []neigh.Neigh{ // IPv4 と IPv6 の 2 行は同じ機器なので、機器は 2 台
			{Dst: "192.0.2.1", Lladdr: "02:00:00:00:00:01", Dev: "wlp2s0"},
			{Dst: "192.0.2.20", Lladdr: "11:22:33:44:55:66", Dev: "wlp2s0"},
			{Dst: "2001:db8:1::1", Lladdr: "02:00:00:00:00:01", Dev: "wlp2s0"},
		},
	}
	var b strings.Builder
	if err := HTML(&b, r, now); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	for _, want := range []string{
		"<title>arch&lt;x&gt; のネットワーク</title>", // ホスト名はエスケープされる
		"取得 2026-09-26 12:34",
		"同じ LAN の機器 2</p>",
		"sudo sanjal --html",
		"図を描画中",
		"font/woff2;base64,",
		"--color-key-50",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("%q が無い", want)
		}
	}
	if strings.Count(s, "</script>") != 2 {
		t.Errorf("</script> が %d 個。mermaid.js と page.js の 2 個のはず", strings.Count(s, "</script>"))
	}
}

func TestHTMLEmpty(t *testing.T) {
	var b strings.Builder
	if err := HTML(&b, hynt.Report{Host: "arch"}, now); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	if !strings.Contains(s, "描けるインタフェースが無い") {
		t.Error("0 件の表示が無い")
	}
	if strings.Contains(s, "<script>") {
		t.Error("描くものが無いのに mermaid.js を埋め込んでいる")
	}
}
