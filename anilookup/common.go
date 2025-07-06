package main

import "github.com/zorchenhimer/go-ed2k"

func hashED2K(data []byte) []byte {
	hashed := ed2k.New()
	hashed.Write(data)
	return hashed.Sum(nil)
}
