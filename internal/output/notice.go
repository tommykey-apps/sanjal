package output

import "github.com/tommykey-apps/hynt"

// deniedParts は root 権限が無くて読めなかったものを、助詞「は」まで付けて返す。無ければ空。
// 欧文の後だけ空白を入れる (「IPsec は」「ファイアウォールは」)
func deniedParts(r hynt.Report) string {
	fw := r.FirewallState == hynt.FirewallDenied
	switch {
	case r.IPsecDenied && fw:
		return "IPsec とファイアウォールは"
	case r.IPsecDenied:
		return "IPsec は"
	case fw:
		return "ファイアウォールは"
	}
	return ""
}

// 入口のポートに「塞がれている」の印が付くのは、ファイアウォールを読めたときだけ
func firewallUnread(r hynt.Report) bool {
	return r.FirewallState == hynt.FirewallDenied || r.FirewallState == hynt.FirewallMissing
}
