package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sthayduk/hibp-checker/internal/hibp"
)

var (
	inputFile  string
	outputFile string
	delimiter  string
	skipHeader bool
	workers    int
	limit      int
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check NTLM hashes against HIBP database",
	Long: `Check NTLM password hashes from a file against the Have I Been Pwned
Pwned Passwords API. The input file should contain lines in the format:
account:hash

Accounts ending with '$' (computer accounts) are automatically skipped.

Results are streamed to the output file as they are found, so partial
results are preserved if the process is interrupted.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if inputFile == "" {
			return fmt.Errorf("input file is required")
		}

		// Open output file early for streaming results
		var resultWriter *hibp.ResultWriter
		if outputFile != "" {
			file, err := os.Create(outputFile)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}
			defer file.Close()

			// Use buffered writer for better performance
			bufferedWriter := bufio.NewWriter(file)
			defer bufferedWriter.Flush()

			resultWriter = hibp.NewResultWriter(bufferedWriter)
			fmt.Printf("Streaming results to: %s\n", outputFile)
		} else {
			resultWriter = hibp.NewResultWriter(nil)
		}

		checker := hibp.NewChecker()

		exposedCount, err := checker.CheckFile(inputFile, delimiter, skipHeader, workers, limit, resultWriter)
		if err != nil {
			return fmt.Errorf("failed to check file: %w", err)
		}

		fmt.Printf("\nTotal exposed accounts: %d\n", exposedCount)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)

	checkCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input file containing account:hash pairs (required)")
	checkCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file for exposed accounts (streamed)")
	checkCmd.Flags().StringVarP(&delimiter, "delimiter", "d", ":", "Delimiter between account and hash")
	checkCmd.Flags().BoolVarP(&skipHeader, "skip-header", "s", false, "Skip the first line (header row)")
	checkCmd.Flags().IntVarP(&workers, "workers", "w", 10, "Number of concurrent workers for API queries")
	checkCmd.Flags().IntVarP(&limit, "limit", "l", 0, "Limit number of accounts to check (0 = no limit)")

	checkCmd.MarkFlagRequired("input")
}
