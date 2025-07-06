package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "anilookup",
	Short: "anilookup is a tool for looking up files in AniDB.",
	Long:  `anilookup can be used to lookup files in AniDB.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
