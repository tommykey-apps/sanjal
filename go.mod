module github.com/tommykey-apps/sanjal

go 1.27.1

require github.com/tommykey-apps/hynt v0.4.1

// 開発機のネットワークの情報をテストに含むため撤回する
retract [v0.1.0, v0.1.2]

// テストに開発機のホスト名とインタフェース名を含むため撤回する
retract [v0.2.0, v0.2.1]
