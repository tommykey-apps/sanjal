// sanjal は、ホストが繋がっているネットワークを VPN を含めて図にする
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tommykey-apps/hynt"
	"github.com/tommykey-apps/sanjal/internal/output"
)

func main() {
	r, err := hynt.Collect(context.Background())
	if err == nil {
		err = output.Markdown(os.Stdout, r)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "sanjal:", err)
		os.Exit(1)
	}
}
