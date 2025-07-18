package anidb

import (
	"bytes"
	"compress/flate"
	"context"
	"crypto/aes"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

type nullHandler struct{}

func (nullHandler) Enabled(context.Context, slog.Level) bool {
	return false
}

func (nullHandler) Handle(context.Context, slog.Record) error {
	return nil
}

func (h nullHandler) WithAttrs([]slog.Attr) slog.Handler {
	return h
}

func (h nullHandler) WithGroup(string) slog.Handler {
	return h
}

var nullLogger = slog.New(nullHandler{})

func TestMux(t *testing.T) {
	t.Parallel()
	ctx := testContext(t, time.Second)
	pc, c := newUDPPipe(t, time.Second)
	m := NewMux(c, nullLogger)
	t.Cleanup(m.Close)

	t.Run("first request", func(t *testing.T) {
		t.Parallel()
		resp, err := m.Request(ctx, "PING", url.Values{"nat": []string{"1"}})
		if err != nil {
			t.Fatal(err)
		}
		want := Response{
			Code:   300,
			Header: "PONG",
			Rows:   [][]string{{"123"}},
		}
		if !reflect.DeepEqual(resp, want) {
			t.Errorf("Got %#v; want %#v", resp, want)
		}
	})
	t.Run("second request", func(t *testing.T) {
		t.Parallel()
		resp, err := m.Request(ctx, "PING", url.Values{})
		if err != nil {
			t.Fatal(err)
		}
		want := Response{
			Code:   300,
			Header: "PONG",
		}
		if !reflect.DeepEqual(resp, want) {
			t.Errorf("Got %#v; want %#v", resp, want)
		}
	})
	t.Run("test server", func(t *testing.T) {
		t.Parallel()
		data := make([]byte, 200)
		var tag1, tag2 responseTag
		for i := 0; i < 2; i++ {
			t.Logf("Reading packet")
			n, _, err := pc.ReadFrom(data)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("Done reading packet")
			tag := parseRequestTag(data[:n])
			if strings.Contains(string(data[:n]), "nat=1") {
				tag1 = tag
			} else {
				tag2 = tag
			}
		}
		addr := c.LocalAddr()
		_, err := pc.WriteTo([]byte(fmt.Sprintf("%s 300 PONG\n123", tag1)), addr)
		if err != nil {
			t.Fatal(err)
		}
		_, err = pc.WriteTo([]byte(fmt.Sprintf("%s 300 PONG", tag2)), addr)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestMux_close_requests(t *testing.T) {
	t.Parallel()
	ctx := testContext(t, time.Second)
	pc, c := newUDPPipe(t, time.Second)
	m := NewMux(c, nullLogger)
	t.Cleanup(m.Close)

	t.Run("first request", func(t *testing.T) {
		t.Parallel()
		_, err := m.Request(ctx, "PING", url.Values{"nat": []string{"1"}})
		if err == nil {
			t.Errorf("Expected error")
		}
	})
	t.Run("second request", func(t *testing.T) {
		t.Parallel()
		_, err := m.Request(ctx, "PING", url.Values{})
		if err == nil {
			t.Errorf("Expected error")
		}
	})
	t.Run("test server", func(t *testing.T) {
		t.Parallel()
		data := make([]byte, 200)
		for i := 0; i < 2; i++ {
			_, _, err := pc.ReadFrom(data)
			if err != nil {
				t.Fatal(err)
			}
		}
		m.Close()
	})
}

// TODO Add test for Mux decryption.

func TestMux_compression(t *testing.T) {
	t.Parallel()
	ctx := testContext(t, time.Second)
	pc, c := newUDPPipe(t, time.Second)
	m := NewMux(c, nullLogger)
	t.Cleanup(m.Close)

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		resp, err := m.Request(ctx, "PING", url.Values{})
		if err != nil {
			t.Fatal(err)
		}
		want := Response{
			Code:   300,
			Header: "PONG",
		}
		if !reflect.DeepEqual(resp, want) {
			t.Errorf("Got %#v; want %#v", resp, want)
		}
	})
	t.Run("test server", func(t *testing.T) {
		t.Parallel()
		data := make([]byte, 200)
		n, _, err := pc.ReadFrom(data)
		if err != nil {
			t.Fatal(err)
		}
		tag := parseRequestTag(data[:n])
		addr := c.LocalAddr()
		resp := []byte(fmt.Sprintf("%s 300 PONG", tag))
		resp = append([]byte{0, 0}, compress(resp)...)
		if _, err := pc.WriteTo(resp, addr); err != nil {
			t.Fatal(err)
		}
	})
}

func TestResponseMap(t *testing.T) {
	t.Parallel()
	t.Run("happy path", func(t *testing.T) {
		t.Parallel()
		m := responseMap{logger: nullLogger}
		ctx := testContext(t, time.Second)
		t.Run("first tag", func(t *testing.T) {
			c := m.waitFor("shefi")
			t.Parallel()
			select {
			case got := <-c:
				const want = "shifuna"
				if string(got) != want {
					t.Errorf("Got %q, want %q", got, want)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		})
		t.Run("second tag", func(t *testing.T) {
			c := m.waitFor("kyaru")
			t.Parallel()
			select {
			case got := <-c:
				const want = "kiruya"
				if string(got) != want {
					t.Errorf("Got %q, want %q", got, want)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		})
		m.deliver("kyaru", []byte("kiruya"))
		m.deliver("shefi", []byte("shifuna"))
	})
	t.Run("close", func(t *testing.T) {
		t.Parallel()
		m := responseMap{logger: nullLogger}
		ctx := testContext(t, time.Second)
		t.Run("first tag", func(t *testing.T) {
			c := m.waitFor("shefi")
			t.Parallel()
			select {
			case got := <-c:
				const want = ""
				if string(got) != want {
					t.Errorf("Got %q, want %q", got, want)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		})
		m.close()
	})
}

func TestParseResponse(t *testing.T) {
	t.Parallel()
	const data = `720 1234 NOTIFICATION - NEW FILE
1234|12|34`
	got, err := parseResponse([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	want := Response{
		Code:   720,
		Header: "1234 NOTIFICATION - NEW FILE",
		Rows: [][]string{
			{"1234", "12", "34"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Got %#v, want %#v", got, want)
	}
}

func TestEncryptDecrypt(t *testing.T) {
	t.Parallel()
	// AES-128, 16 bytes
	const key = "\x80\xa2_\xcaa\xb6\f\xa9X\xa5\xff\x9am\xebי"
	cb, err := aes.NewCipher([]byte(key))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		desc string
		size int
	}{
		{desc: "3 bytes", size: 3},
		{desc: "16 bytes", size: 16},
		{desc: "17 bytes", size: 17},
		{desc: "31 bytes", size: 31},
		{desc: "32 bytes", size: 32},
		{desc: "33 bytes", size: 33},
		{desc: "64 bytes", size: 64},
	}
	for _, c := range cases {
		c := c
		t.Run(c.desc, func(t *testing.T) {
			t.Parallel()
			orig := make([]byte, c.size)
			if _, err := rand.Read(orig); err != nil {
				t.Fatal(err)
			}
			data := make([]byte, len(orig))
			copy(data, orig)
			data = encrypt(cb, data)
			if reflect.DeepEqual(orig, data) {
				t.Fatalf("data not encrypted")
			}
			t.Logf("encrypted data is %d bytes", len(data))
			data, err = decrypt(cb, data)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(orig, data) {
				t.Errorf("decrypted not equal, got %x, want %x", data, orig)
			}
		})
	}
}

var tagRegexp = regexp.MustCompile(`tag=([0-9]+)`)

func parseRequestTag(b []byte) responseTag {
	m := tagRegexp.FindSubmatch(b)
	return responseTag(m[1])
}

// DEFLATE
func compress(b []byte) []byte {
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, 3)
	if err != nil {
		panic(err)
	}
	defer w.Close()
	if _, err := w.Write(b); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func newUDPPipe(t *testing.T, timeout time.Duration) (net.PacketConn, net.Conn) {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pc.Close() })
	if err := pc.SetDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatal(err)
	}
	c, err := net.Dial("udp", pc.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	if err := c.SetDeadline(time.Now().Add(timeout)); err != nil {
		t.Fatal(err)
	}
	return pc, c
}

func testContext(t *testing.T, timeout time.Duration) context.Context {
	ctx, cf := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cf)
	return ctx
}
