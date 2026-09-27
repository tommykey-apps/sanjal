// Package output は図を Markdown / HTML の文書に包む
package output

import (
	"fmt"
	"io"

	"github.com/tommykey-apps/hynt"
	"github.com/tommykey-apps/sanjal/internal/diagram"
)

// Mermaid 自体に凡例の機能は無いので、図の下に表で置く
const legendMarkdown = `| 形 | 意味 |
|---|---|
| 六角形 | このホスト |
| 枠 | 区分。機械の中 (外へ出ない docker などの口)、LAN、VPN |
| 丸 | インターネット。ここへ矢印が届く口が、普段の通信の出口 |
| 四角 | インタフェース。名前、種別、アドレス。止まっていれば (DOWN) |
| 両端の丸い枠 | 経路の宛先。` + "`via`" + ` の後ろはゲートウェイ。` + "`(使われていない)`" + ` の既定経路は VPN など別の経路が優先されている |
| 角丸の四角 | 同じ LAN の機器 (最近通信した相手)。機器ごとに IP と MAC。「ゲートウェイ」はインターネットへの中継役 |
| 実線 | 物理インタフェース、または main テーブルの経路 |
| 矢印の ` + "`tun`" + ` ` + "`wireguard`" + ` ` + "`ppp`" + ` | VPN の実装 |
| 矢印の ` + "`table 52`" + ` など | main 以外のルーティングテーブルの経路 |
| 矢印の ` + "`ipsec via`" + ` | インタフェースを持たない IPsec |
| 破線 | 同じ LAN の機器 |
`

func Markdown(w io.Writer, r hynt.Report) error {
	_, err := fmt.Fprintf(w, "# %s のネットワーク\n\n```mermaid\n%s```\n\n## 凡例\n\n%s", r.Host, diagram.Mermaid(r), legendMarkdown)
	if err != nil {
		return err
	}
	if r.IPsecDenied {
		_, err = fmt.Fprintln(w, "\nIPsec は root 権限が無いため読めていない。`sudo sanjal` で再実行すると描かれる。")
	}
	return err
}
