// sanjal は、ホストが繋がっているネットワークを VPN を含めて図にする
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/tommykey-apps/hynt"
	"github.com/tommykey-apps/sanjal/internal/output"
)

// GoReleaser が -X main.version=1.2.3 で埋める。埋めていなければ dev
var version = "dev"

// go install で入れたときは -X が渡らないので、Go が記録したモジュールの版で補う。
// GoReleaser の表記 (1.2.3) に揃えて先頭の v を落とす。go run では (devel) なので dev のまま
func versionString(v string, bi *debug.BuildInfo, ok bool) string {
	if v != "dev" || !ok || bi.Main.Version == "" || bi.Main.Version == "(devel)" {
		return v
	}
	return strings.TrimPrefix(bi.Main.Version, "v")
}

func main() {
	asHTML := flag.Bool("html", false, "mermaid.js を埋め込んだ 1 枚の HTML で出す")
	showVersion := flag.Bool("version", false, "版を出す")
	flag.Parse()
	if *showVersion {
		bi, ok := debug.ReadBuildInfo()
		fmt.Println("sanjal", versionString(version, bi, ok))
		return
	}

	r, err := hynt.Collect(context.Background())
	if err == nil {
		if *asHTML {
			err = output.HTML(os.Stdout, r, time.Now())
		} else {
			err = output.Markdown(os.Stdout, r)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "sanjal:", err)
		os.Exit(1)
	}
}
