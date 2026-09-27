// sanjal は、ホストが繋がっているネットワークを VPN を含めて図にする
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/tommykey-apps/hynt"
	"github.com/tommykey-apps/sanjal/internal/diagram"
)

func main() {
	r, err := hynt.Collect(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "sanjal:", err)
		os.Exit(1)
	}
	// Markdown のコードブロックで囲むと、GitHub や VS Code がそのまま図にする
	fmt.Print("```mermaid\n" + diagram.Mermaid(r) + "```\n")
}
