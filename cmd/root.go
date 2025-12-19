package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "hibp-checker",
	Short: "Check NTLM password hashes against Have I Been Pwned",
	Long: `hibp-checker is a CLI tool that checks NTLM password hashes
against the Have I Been Pwned Pwned Passwords API.

It reads a file containing account:hash pairs and identifies
which passwords have been exposed in data breaches.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
