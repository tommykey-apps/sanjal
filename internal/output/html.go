package output

import (
	"bytes"
	"embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"

	"github.com/tommykey-apps/hynt"
	"github.com/tommykey-apps/hynt/neigh"
	"github.com/tommykey-apps/sanjal/internal/diagram"
)

// Why not: 別ファイルを並べない。1 枚の HTML にすべて埋め込み、ネットに繋がらなくても
// ファイルを開くだけで見えるようにする。代わりに HTML は約 7MB になる
//
//go:embed assets
var assets embed.FS

var page = template.Must(template.ParseFS(assets, "assets/page.html"))

// fontFaces は Noto Sans JP (本文) と Noto Sans Mono (Mermaid の記述) を data URI で読む
var fontFaces = []struct {
	family, file, weight, unicodeRange string
}{
	{"Noto Sans JP", "noto-sans-jp-latin-400-normal.woff2", "400", "U+0000-00FF, U+0131, U+0152-0153, U+02BB-02BC, U+02C6, U+02DA, U+02DC, U+0304, U+0308, U+0329, U+2000-206F, U+2074, U+20AC, U+2122, U+2191, U+2193, U+2212, U+2215, U+FEFF, U+FFFD"},
	{"Noto Sans JP", "noto-sans-jp-latin-700-normal.woff2", "700", "U+0000-00FF, U+0131, U+0152-0153, U+02BB-02BC, U+02C6, U+02DA, U+02DC, U+0304, U+0308, U+0329, U+2000-206F, U+2074, U+20AC, U+2122, U+2191, U+2193, U+2212, U+2215, U+FEFF, U+FFFD"},
	{"Noto Sans JP", "noto-sans-jp-japanese-400-normal.woff2", "400", ""},
	{"Noto Sans JP", "noto-sans-jp-japanese-700-normal.woff2", "700", ""},
	{"Noto Sans Mono", "noto-sans-mono-latin-400-normal.woff2", "400", ""},
}

type pageData struct {
	Host                  string
	Generated             string
	Links, Routes, Neighs int
	IPsecDenied, Empty    bool
	Source                string
	CSS                   template.CSS
	MermaidJS, PageJS     template.JS
}

func HTML(w io.Writer, r hynt.Report, now time.Time) error {
	css, err := styles()
	if err != nil {
		return err
	}
	mermaidJS, err := assets.ReadFile("assets/mermaid.min.js")
	if err != nil {
		return err
	}
	pageJS, err := assets.ReadFile("assets/page.js")
	if err != nil {
		return err
	}
	src := diagram.Mermaid(r)
	d := pageData{
		Host:        r.Host,
		Generated:   now.Format("2006-01-02 15:04"),
		Links:       len(r.Links),
		Routes:      len(r.Routes),
		Neighs:      countDevices(r.Neighs),
		IPsecDenied: r.IPsecDenied,
		// host の 1 行と graph LR だけなら、描くものが無い
		Empty:     strings.Count(src, "\n") <= 2,
		Source:    src,
		CSS:       template.CSS(css),
		MermaidJS: template.JS(mermaidJS),
		PageJS:    template.JS(pageJS),
	}
	var buf bytes.Buffer
	if err := page.Execute(&buf, d); err != nil {
		return err
	}
	_, err = buf.WriteTo(w)
	return err
}

// 同じ機器は IPv4 と IPv6 で別の行として来るので、行数ではなく MAC で数える
func countDevices(ns []neigh.Neigh) int {
	n := 0
	for _, c := range neigh.CountByDev(ns) {
		n += c
	}
	return n
}

// tokens.css (トークン) + dark.css (暗色の上書き) + 書体 + page.css (この画面) の順に連結する
func styles() (string, error) {
	var b strings.Builder
	for _, name := range []string{"tokens.css", "dark.css"} {
		c, err := assets.ReadFile("assets/" + name)
		if err != nil {
			return "", err
		}
		b.Write(c)
		b.WriteByte('\n')
	}
	for _, f := range fontFaces {
		data, err := assets.ReadFile("assets/fonts/" + f.file)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "@font-face{font-family:%q;font-weight:%s;font-style:normal;font-display:swap;src:url(data:font/woff2;base64,%s) format(\"woff2\");",
			f.family, f.weight, base64.StdEncoding.EncodeToString(data))
		if f.unicodeRange != "" {
			fmt.Fprintf(&b, "unicode-range:%s;", f.unicodeRange)
		}
		b.WriteString("}\n")
	}
	c, err := assets.ReadFile("assets/page.css")
	if err != nil {
		return "", err
	}
	b.Write(c)
	return b.String(), nil
}
