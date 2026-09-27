package output

import (
	"strings"
	"testing"

	"github.com/tommykey-apps/hynt"
)

func TestMarkdownIPsecDenied(t *testing.T) {
	var b strings.Builder
	if err := Markdown(&b, hynt.Report{Host: "arch", IPsecDenied: true}); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	for _, want := range []string{"# arch のネットワーク", "```mermaid\ngraph LR\n", "## 凡例", "`sudo sanjal`"} {
		if !strings.Contains(s, want) {
			t.Errorf("%q が無い:\n%s", want, s)
		}
	}
}

func TestMarkdownIPsecRead(t *testing.T) {
	var b strings.Builder
	if err := Markdown(&b, hynt.Report{Host: "arch"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "root 権限") {
		t.Errorf("読めているのに権限の注意が出る:\n%s", b.String())
	}
}
