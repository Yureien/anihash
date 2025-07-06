package main

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	enableMD5  bool
	enableSHA1 bool
)

func init() {
	rootCmd.AddCommand(hashCmd)
	hashCmd.Flags().BoolVar(&enableMD5, "md5", false, "Enable MD5 hashing")
	hashCmd.Flags().BoolVar(&enableSHA1, "sha1", false, "Enable SHA1 hashing")
}

var hashCmd = &cobra.Command{
	Use:   "hash [file]",
	Short: "Get hashes for a file",
	Long:  `Get hashes for a file. ed2k is always calculated.`,
	Args:  cobra.ExactArgs(1),
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

		ed2kStart := time.Now()
		ed2kHash := hashED2K(fileBytes)
		ed2kDuration := time.Since(ed2kStart)
		fmt.Printf("ed2k: %s (took %s)\n", hex.EncodeToString(ed2kHash), ed2kDuration)

		if enableMD5 {
			md5Start := time.Now()
			md5Hash := md5.Sum(fileBytes)
			md5Duration := time.Since(md5Start)
			fmt.Printf("md5:  %s (took %s)\n", hex.EncodeToString(md5Hash[:]), md5Duration)
		}

		if enableSHA1 {
			sha1Start := time.Now()
			sha1Hash := sha1.Sum(fileBytes)
			sha1Duration := time.Since(sha1Start)
			fmt.Printf("sha1: %s (took %s)\n", hex.EncodeToString(sha1Hash[:]), sha1Duration)
		}

		fmt.Printf("file size: %d bytes\n", len(fileBytes))
	},
}
