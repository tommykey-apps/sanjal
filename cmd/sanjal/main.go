// sanjal は、ホストが繋がっているネットワークを VPN を含めて図にする
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/tommykey-apps/hynt"
	"github.com/tommykey-apps/sanjal/internal/output"
)

// GoReleaser が -X main.version=v1.2.3 で埋める。go run のときは dev
var version = "dev"

func main() {
	asHTML := flag.Bool("html", false, "mermaid.js を埋め込んだ 1 枚の HTML で出す")
	showVersion := flag.Bool("version", false, "版を出す")
	flag.Parse()
	if *showVersion {
		fmt.Println("sanjal", version)
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
