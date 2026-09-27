# sanjal

ホストが繋がっているネットワークを、VPN を含めて図にする CLI。

    sanjal > net.md        # Mermaid 入りの Markdown
    sanjal --html > net.html

情報の収集は [hynt](https://github.com/tommykey-apps/hynt) をライブラリとして使う。仕様は `docs/SPEC.md`。

## 導入

Linux (amd64 / arm64)。Releases から tar.gz を取って置く:

    curl -L https://github.com/tommykey-apps/sanjal/releases/latest/download/sanjal_$(curl -s https://api.github.com/repos/tommykey-apps/sanjal/releases/latest | grep -Po '"tag_name": "v\K[^"]+')_linux_amd64.tar.gz | tar xz sanjal
    sudo install -Dm755 sanjal /usr/local/bin/sanjal

Go が入っていれば:

    go install github.com/tommykey-apps/sanjal/cmd/sanjal@latest

iproute2 (`ip`) が要る。IPsec を描くときだけ root が要る。
