package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var (
	anihashURL string
)

func init() {
	rootCmd.AddCommand(lookupCmd)
	lookupCmd.Flags().StringVar(&anihashURL, "url", "http://localhost:8080", "The URL of the anihash server")
}

var lookupCmd = &cobra.Command{
	Use:   "lookup [file]",
	Short: "Lookup a file from anihash",
	Long: `
Lookup a file from anihash. This will first generate an ED2K hash, then query anihash for the file.
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

		ed2kHash := hashED2K(fileBytes)
		ed2kHashStr := hex.EncodeToString(ed2kHash)

		resp, err := http.Get(fmt.Sprintf("%s/query/ed2k?size=%d&ed2k=%s", anihashURL, len(fileBytes), ed2kHashStr))
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
