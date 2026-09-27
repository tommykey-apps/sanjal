package main

import (
	"runtime/debug"
	"testing"
)

func TestVersionString(t *testing.T) {
	info := func(v string) *debug.BuildInfo {
		return &debug.BuildInfo{Main: debug.Module{Version: v}}
	}
	for _, c := range []struct {
		name, v string
		bi      *debug.BuildInfo
		ok      bool
		want    string
	}{
		{"GoReleaser が埋めた版を優先する", "0.1.0", info("v0.1.0"), true, "0.1.0"},
		{"go install はモジュールの版", "dev", info("v0.1.0"), true, "0.1.0"},
		{"go run は (devel) なので dev", "dev", info("(devel)"), true, "dev"},
		{"版が空なら dev", "dev", info(""), true, "dev"},
		{"ビルド情報が無ければ dev", "dev", nil, false, "dev"},
	} {
		if got := versionString(c.v, c.bi, c.ok); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
