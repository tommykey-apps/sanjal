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

- host を中心にした星形。host → インタフェース → 宛先 (経路) / 隣人 の順に外へ広がる
- インタフェースは四角、宛先は円、隣人は四角に IP と MAC。隣人への線は破線
- VPN のインタフェースは矢印に実装名 (tun / wireguard / ppp) を書く。物理は矢印なしの線
- main 以外のテーブルの経路は矢印にテーブル名を書く (`table 52`)
- インタフェースを持たない IPsec は host から直接、相手側の範囲へ `ipsec via <gateway>` の矢印を引く。root でなく読めなければ「IPsec は権限なしで未取得」と図の下に 1 行書く
- アドレスも経路も無い仮想インタフェース (veth) は描かない
- ラベルの改行は `<br/>`。`\n` は引用符の中でも改行にならない
- ノード ID に使えるのは英数字と `_` だけ。名前の他の文字は `_` に落とす
- 名前でソートしてから描く。出力を決定的にするため
- 凡例は図の外に置く。Markdown では図の下の表、HTML では図の上の一覧 (ahab と同じ方式)

## 技術方針

- 外部モジュールは hynt だけ
- CLI は `cmd/sanjal`
- Mermaid の文字列生成は hynt の `Report` を受け取る純粋な関数にし、記録済みの `Report` でテストする
- HTML は `html/template` と `go:embed`。mermaid.js は ahab と同じく同梱し、ネットに繋がらなくても開ける
- cgo なし。単一バイナリ

## 配布

GitHub Releases (GoReleaser、linux amd64 / arm64)。副で `go install github.com/tommykey-apps/sanjal/cmd/sanjal@latest`。
hynt は Go のモジュールとして取り込んで同梱するので、利用者が hynt を別に入れる必要はない。
