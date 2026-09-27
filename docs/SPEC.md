# sanjal

ホストが繋がっているネットワークを、**VPN を含めて**図にする CLI。

名前はヒンディー語 संजाल (sanjāl、ネットワーク)。

## hynt との分担

| | hynt | sanjal |
|---|---|---|
| 役割 | 集める。一覧を表で出す | 描く |
| 収集処理 | 持つ (`hynt.Collect`) | 持たない。hynt を import する |
| 出力 | 表 / JSON | Mermaid 入り Markdown / HTML |

sanjal は `ip` コマンドも `/sys` も直接読まない。必要な情報が hynt に無ければ、hynt 側に足す。

## 前提

Linux のみ。iproute2 (`ip`) が入っていること (hynt の前提をそのまま引き継ぐ)。root 不要。root でなければ IPsec だけ描かれない。

## やらないこと

収集処理の実装、常駐、Web サーバー、自動更新、図の編集、VPN の向こう側の機器列挙、Linux 以外の OS。

## 起動

```
sanjal                   # Mermaid 入りの Markdown を標準出力へ
sanjal --html            # mermaid.js を埋め込んだ 1 枚の HTML を標準出力へ
```

開発時は `go run ./cmd/sanjal`。

## 図のルール

- host から右へ広がる図 (`graph LR`)。host → インタフェース → 宛先 (経路) / 同じ LAN の機器 → インターネット
- インタフェースを 3 つの枠 (subgraph) に分ける。上から LAN、VPN、機械の中 (外へ出ない)。名前では決めない
  - VPN: hynt の種別が vpn
  - LAN: 種別が ethernet / wifi。または所属する口 (`Link.Master`) に ethernet / wifi を持つ (br0 や bond)。または、ゲートウェイ付きの既定経路を持つ (vlan など)
  - 機械の中: それ以外 (docker0 などのブリッジ)
- host は枠に入れない。入れると線が枠をまたいで交差し、読めなくなった
- インターネットは丸 1 つ。普通の通信 (fwmark なし) が使う既定経路だけをそこへ繋ぐ
  - `ip rule` を優先度の順にたどり、既定経路を持つ最初のテーブルで決める。当てはまる規則は `from all` と `not from all fwmark ...`。fwmark 付きの規則と unreachable などは飛ばす
  - `suppress_prefixlength N` の規則では、長さ N 以下の経路を無視する (wg-quick)
  - 既定経路は `default` と `0.0.0.0/1` `128.0.0.0/1` (IPv6 は `::/1` `8000::/1`)。同じテーブルに両方あれば /1 だけ使う (OpenVPN)
  - rule が無い Report では main だけを見る
  - ゲートウェイ付きなら インタフェース → ゲートウェイ → インターネット。ゲートウェイが同じ LAN の機器にいれば、そのノードを兼ねて「ゲートウェイ」と書く。IPv6 の fe80:: のゲートウェイは EUI-64 から MAC を戻して照合する
  - インタフェースから直接インターネットへ引く線には table 名を書かない。VPN の枠の見出しに重なる
  - 選ばれなかった既定経路は宛先の枠に `(使われていない)` と書く
- 形: host は六角形、インタフェースは四角、宛先は両端の丸い枠、同じ LAN の機器は角丸の四角、インターネットは丸
- ブリッジや bond に属する口は、host ではなく親のインタフェースから線を引く
- VPN のインタフェースは host からの矢印に実装名 (tun / wireguard / ppp) を書く。物理は矢印なしの線
- main 以外のテーブルの経路は矢印にテーブル名を書く (`table 52`)
- 止まっているインタフェースはラベルに `(DOWN)` を足す。UNKNOWN は書かない (tun は常に UNKNOWN)
- 同じ LAN の機器 (`ip neigh`) は破線。同じ機器は IPv4 と IPv6 で別の行として来るので、MAC ごとに 1 つのノードにまとめて IP を並べる
- インタフェースを持たない IPsec は host から直接、相手側の範囲へ `ipsec via <gateway>` の矢印を引く。root でなく読めなければ、その旨を図の外に 1 行書く
- アドレスも経路も無い仮想インタフェース (veth) は描かない。描かなかったインタフェースにいる同じ LAN の機器も描かない
- ラベルの改行は `<br/>`。引用符は `#quot;`。ノード ID は英数字と `_` だけ
- 並べ替えない。hynt が名前でソートして返すので、同じ Report なら同じ図になる
- 凡例は図の外に置く。Markdown では図の下の表、HTML では図の上の一覧 (ahab と同じ方式)

## HTML の画面

- 1 枚の HTML に mermaid.js、デザイントークン、書体を埋め込む。ネットに繋がらなくても開ける。大きさは約 6.4MB
- 見た目はデジタル庁デザインシステムのトークンだけで組む。`tokens.css`、書体、暗色の上書きは ahab から複製する
- 明暗は OS の設定に従い、切り替わったら描き直す
- 状態: 描画中、失敗 (理由と mermaid.live への案内)、0 件 (`ip -br addr` の案内)、権限なし (IPsec の注意)
- 図は原寸で置き、狭い画面では図の枠の中だけ横にスクロールする

## 技術方針

- 外部モジュールは hynt だけ
- CLI は `cmd/sanjal`
- Mermaid の文字列生成 (`internal/diagram`) は hynt の `Report` を受け取る純粋な関数にし、記録済みの `Report` でテストする
- hynt は `go get` で版を固定して取り込む。hynt を直しながら試すときはコミットしない `go.work` を使う
- HTML は `html/template` と `go:embed`。mermaid.js は ahab と同じく同梱し、ネットに繋がらなくても開ける
- cgo なし。単一バイナリ

## 配布

GitHub Releases (GoReleaser、linux amd64 / arm64)。バイナリは約 10MB。副で `go install github.com/tommykey-apps/sanjal/cmd/sanjal@latest`。
hynt は Go のモジュールとして取り込んで同梱するので、利用者が hynt を別に入れる必要はない。
