package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(shaLookupCmd)
	shaLookupCmd.Flags().StringVar(&anihashURL, "url", "https://anihash.sohamsen.me", "The URL of the anihash server")
}

var shaLookupCmd = &cobra.Command{
	Use:   "sha-lookup [file]",
	Short: "Lookup a file from anihash using SHA1",
	Long: `
Lookup a file from anihash using SHA1. This will first generate a SHA1 hash, then query anihash for the file.
This is faster than ED2K hashing, but will only work if the file is present in the anihash database.
`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		file, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("Error opening file %s: %v\n", filePath, err)
			os.Exit(1)
		}
		defer file.Close()

		fileBytes, err := io.ReadAll(file)
		if err != nil {
			fmt.Printf("Error reading file %s: %v\n", filePath, err)
			os.Exit(1)
		}

		sha1Hash := sha1.Sum(fileBytes)
		sha1HashStr := hex.EncodeToString(sha1Hash[:])

		resp, err := http.Get(fmt.Sprintf("%s/query/hash?hash=%s", anihashURL, sha1HashStr))
		if err != nil {
			fmt.Printf("Error querying anihash: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error reading response body: %v\n", err)
			os.Exit(1)
		}

		var response map[string]any
		if err := json.Unmarshal(body, &response); err != nil {
			fmt.Printf("Error unmarshalling response: %v\n", err)
			os.Exit(1)
		}

		prettyJSON, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			fmt.Printf("Error marshalling response: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(prettyJSON))
	},
}
