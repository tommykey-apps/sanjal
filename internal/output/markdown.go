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
| 四角 | インタフェース。名前、種別、アドレス。止まっていれば (DOWN) |
| 両端の丸い枠 | 経路の宛先。` + "`via`" + ` の後ろはゲートウェイ |
| 角丸の四角 | 隣人 (同じ LAN で最近通信した相手)。機器ごとに IP と MAC |
| 実線 | 物理インタフェース、または main テーブルの経路 |
| 矢印の ` + "`tun`" + ` ` + "`wireguard`" + ` ` + "`ppp`" + ` | VPN の実装 |
| 矢印の ` + "`table 52`" + ` など | main 以外のルーティングテーブルの経路 |
| 矢印の ` + "`ipsec via`" + ` | インタフェースを持たない IPsec |
| 破線 | 隣人 |
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
