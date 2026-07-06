package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd 显示由 GoReleaser / Makefile 通过 -ldflags 注入的版本信息。
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("eastmoney version %s (commit: %s, built: %s)\n", version, commit, date)
	},
}
