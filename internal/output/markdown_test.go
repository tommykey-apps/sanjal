package output

import (
	"strings"
	"testing"

	"github.com/tommykey-apps/hynt"
)

func TestMarkdownIPsecDenied(t *testing.T) {
	var b strings.Builder
	if err := Markdown(&b, hynt.Report{Host: "box", IPsecDenied: true}); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	for _, want := range []string{"# box のネットワーク", "```mermaid\ngraph LR\n", "## 凡例", "`sudo sanjal`"} {
		if !strings.Contains(s, want) {
			t.Errorf("%q が無い:\n%s", want, s)
		}
	}
}

func TestMarkdownIPsecRead(t *testing.T) {
	var b strings.Builder
	if err := Markdown(&b, hynt.Report{Host: "box"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "root 権限") {
		t.Errorf("読めているのに権限の注意が出る:\n%s", b.String())
	}
}

func TestMarkdownFirewallNotices(t *testing.T) {
	cases := []struct {
		state       string
		ipsecDenied bool
		want        string // 凡例の表の後ろに出る注意書き
	}{
		{hynt.FirewallDenied, true, "IPsec とファイアウォールは root 権限が無いため読めていない。`sudo sanjal` で再実行すると描かれる。入口のポートがファイアウォールで塞がれているかは分からない。"},
		{hynt.FirewallDenied, false, "ファイアウォールは root 権限が無いため読めていない。`sudo sanjal` で再実行すると描かれる。入口のポートがファイアウォールで塞がれているかは分からない。"},
		{hynt.FirewallMissing, false, "nft が無いため、ファイアウォールを読めていない。入口のポートがファイアウォールで塞がれているかは分からない。"},
		{hynt.FirewallRead, false, ""},
	}
	for _, c := range cases {
		var b strings.Builder
		if err := Markdown(&b, hynt.Report{Host: "box", IPsecDenied: c.ipsecDenied, FirewallState: c.state}); err != nil {
			t.Fatal(err)
		}
		s := b.String()
		got := strings.TrimSpace(s[strings.LastIndex(s, "|\n")+2:])
		if got != c.want {
			t.Errorf("%s: got %q\nwant %q", c.state, got, c.want)
		}
	}
}
