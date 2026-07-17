// Command assaio-agent generates offline token/cost reports from local AI-coding session logs.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/assaio/assaio/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
