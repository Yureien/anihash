package anidb

import (
	"fmt"
	"strings"
)

// A bitSpec designates a bit in an API mask.
type bitSpec struct {
	byte uint8
	bit  uint8
	typ  string
}

// A FileFmask is a mask for the FILE command fmask field.
type FileFmask [5]byte

// FileFmaskFields describes the bit fields in a FILE fmask.
var FileFmaskFields = map[string]bitSpec{
	// byte 0
	"aid":   {0, 6, "int4"},
	"eid":   {0, 5, "int4"},
	"gid":   {0, 4, "int4"},
	"state": {0, 0, "int2"},

	// byte 1
	"size": {1, 3, "int8"},
	"ed2k": {1, 2, "str"},
	"md5":  {1, 1, "str"},
	"sha1": {1, 0, "str"},
	"crc":  {1, 7, "str"},

	// byte 2
	"quality":         {2, 7, "str"},
	"source":          {2, 6, "str"},
	"audio codec":     {2, 5, "str"},
	"audio bitrate":   {2, 4, "int4"},
	"video codec":     {2, 3, "str"},
	"video bitrate":   {2, 2, "int4"},
	"video res":       {2, 1, "str"},
	"video extension": {2, 0, "str"},
}

// Set sets a bit in the mask.
// See [FileFmaskFields] for the field names.
func (m *FileFmask) Set(f ...string) {
	for _, f := range f {
		setMaskBit(m[:], FileFmaskFields, f)
	}
}

// A FileAmask is a mask for the FILE command amask field.
type FileAmask [4]byte

// FileAmaskFields describes the bit fields in a FILE amask.
var FileAmaskFields = map[string]bitSpec{
	// byte 0
	"year": {0, 5, "str"},
	"type": {0, 4, "str"},

	// byte 1
	"romaji name":  {1, 7, "str"},
	"english name": {1, 5, "str"},

	// byte 2
	"epno":           {2, 7, "str"},
	"ep name":        {2, 6, "str"},
	"ep romaji name": {2, 5, "str"},

	// byte 3
	"group name": {3, 7, "str"},
}

// Set sets a bit in the mask.
// See [FileAmaskFields] for the field names.
func (m *FileAmask) Set(f ...string) {
	for _, f := range f {
		setMaskBit(m[:], FileAmaskFields, f)
	}
}

func setMaskBit(b []byte, m map[string]bitSpec, name string) {
	s, ok := m[name]
	if !ok {
		panic(name)
	}
	b[s.byte] |= 1 << s.bit
}

func formatMask(m []byte) string {
	var sb strings.Builder
	for _, b := range m {
		fmt.Fprintf(&sb, "%02x", b)
	}
	return sb.String()
}
