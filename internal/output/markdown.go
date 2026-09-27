// Package output は図を Markdown / HTML の文書に包む
package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/tommykey-apps/hynt"
	"github.com/tommykey-apps/sanjal/internal/diagram"
)

// Mermaid 自体に凡例の機能は無いので、図の下に表で置く
const legendMarkdown = `| 形 | 意味 |
|---|---|
| 六角形 | このホスト |
| 枠 | 区分。マシンの中 (外へ出ない docker などの口)、LAN、VPN |
| 丸 | インターネット。ここへ矢印が届く口が、普段の通信の出口 |
| 四角 | インタフェース。名前、種別、アドレス。止まっていれば (DOWN) |
| 両端の丸い枠 | 経路の宛先。` + "`via`" + ` の後ろはゲートウェイ。` + "`(使われていない)`" + ` の既定経路は VPN など別の経路が優先されている |
| 角丸の四角 | 同じ LAN の機器 (最近通信した相手)。機器ごとに IP と MAC。「ゲートウェイ」はインターネットへの中継役 |
| 実線 | 物理インタフェース、または main テーブルの経路 |
| 矢印の ` + "`tun`" + ` ` + "`wireguard`" + ` ` + "`ppp`" + ` | VPN の実装 |
| 矢印の ` + "`table 52`" + ` など | main 以外のルーティングテーブルの経路 |
| 矢印の ` + "`ipsec via`" + ` | インタフェースを持たない IPsec |
| 破線 | 同じ LAN の機器 |
| 平行四辺形 | 入口。外から入ってこられる (待ち受けている) ポート。「塞がれている」の下はファイアウォールが塞ぐ、「条件付き」の下は送信元などで一部だけ通す |
| 破線の矢印の「受信」 | 入口から口へ入ってくる向き |
| 「マシンの中からだけ」 | 127.0.0.1 などで待ち受け、外からは入れないポート |
`

func Markdown(w io.Writer, r hynt.Report) error {
	_, err := fmt.Fprintf(w, "# %s のネットワーク\n\n```mermaid\n%s```\n\n## 凡例\n\n%s", r.Host, diagram.Mermaid(r), legendMarkdown)
	if err != nil {
		return err
	}
	var notes []string
	if d := deniedParts(r); d != "" {
		notes = append(notes, d+" root 権限が無いため読めていない。`sudo sanjal` で再実行すると描かれる。")
	}
	if r.FirewallState == hynt.FirewallMissing {
		notes = append(notes, "nft が無いため、ファイアウォールを読めていない。")
	}
	if firewallUnread(r) {
		notes = append(notes, "入口のポートがファイアウォールで塞がれているかは分からない。")
	}
	if len(notes) > 0 {
		_, err = fmt.Fprintf(w, "\n%s\n", strings.Join(notes, ""))
	}
	return err
}
