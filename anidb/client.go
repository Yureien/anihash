package anidb

import (
	"context"
	"crypto/aes"
	"crypto/md5"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"
)

const protoVer = "3"

// A Client is an AniDB UDP API client.
//
// The client handles rate limiting.
// The client does not handle retries.
// The client does not handle keepalive.
type Client struct {
	conn    net.Conn
	m       *Mux
	limiter *limiter
	logger  *slog.Logger

	sessionKey syncVar[string]

	ClientName    string
	ClientVersion int32
}

// NewAuthenticatedClient creates a new authenticated AniDB client.
// It returns the client, a function to logout and an error.
// The function to logout should be called when the client is no longer needed.
// The function to logout will return an error if the logout fails.
// The client will be closed when the function to logout is called.
// The client will be authenticated with the given configuration.
// The client will be connected to the given address.
func NewAuthenticatedClient(l *slog.Logger, cfg *AniDBConfig) (*Client, func() error, error) {
	client, err := Dial(cfg.Address, l, cfg.ClientName, cfg.ClientVersion)
	if err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("udpapi NewAuthenticatedClient: %w", err)
	}

	_, err = client.Auth(context.Background(), UserInfo{
		UserName:     cfg.User,
		UserPassword: cfg.Password,
	})
	if err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("udpapi NewAuthenticatedClient: %w", err)
	}

	closeFunc := func() error {
		defer client.Close()

		err := client.Logout(context.Background())
		if err != nil {
			l.Error("failed to logout", "error", err)
		}
		return err
	}

	return client, closeFunc, nil
}

// Dial connects to an AniDB UDP API server.
// The caller should call [Client.SetLogger] as the client may produce
// asynchronous errors.
func Dial(addr string, l *slog.Logger, name string, version int32) (*Client, error) {
	conn, err := net.Dial("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("udpapi NewClient: %w", err)
	}
	l = l.With("package", "go.felesatra.moe/anidb/udpapi", "component", "client")
	c := &Client{
		conn:          conn,
		m:             NewMux(conn, l),
		limiter:       newLimiter(),
		logger:        l,
		ClientName:    name,
		ClientVersion: version,
	}
	return c, nil
}

// LocalPort returns the local port for the client connection.
// This is useful for detecting NAT.
func (c *Client) LocalPort() string {
	addr := c.conn.LocalAddr().String()
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		panic(err)
	}
	return port
}

// Close closes the Client.
// This does not call LOGOUT, so you should try to LOGOUT first.
// The underlying connection is closed.
// No new requests will be accepted (as the connection is closed).
// Outstanding requests will be unblocked.
func (c *Client) Close() {
	// The connection is closed by the Mux.
	c.m.Close()
}

// A UserInfo contains user information for authentication and encryption.
type UserInfo struct {
	UserName     string
	UserPassword string
	APIKey       string // required for encryption, optional otherwise
}

// Encrypt calls the ENCRYPT command.
func (c *Client) Encrypt(ctx context.Context, u UserInfo) error {
	if u.APIKey == "" {
		return errors.New("udpapi encrypt: APIKey required for encryption")
	}
	v := url.Values{}
	v.Set("user", u.UserName)
	v.Set("type", "1")
	resp, err := c.request(ctx, "ENCRYPT", v)
	if err != nil {
		return fmt.Errorf("udpapi Encrypt: %s", err)
	}
	switch resp.Code {
	case 209:
		parts := strings.SplitN(resp.Header, " ", 2)
		salt := parts[0]
		sum := md5.Sum([]byte(u.APIKey + salt))
		b, err := aes.NewCipher(sum[:])
		if err != nil {
			return fmt.Errorf("udpapi Encrypt: %s", err)
		}
		c.m.SetBlock(b)
		return nil
	default:
		return fmt.Errorf("udpapi Encrypt: bad code %d %q", resp.Code, resp.Header)
	}
}

// Auth calls the AUTH command.
func (c *Client) Auth(ctx context.Context, u UserInfo) (port string, _ error) {
	v := url.Values{}
	v.Set("user", u.UserName)
	v.Set("pass", u.UserPassword)
	v.Set("protover", protoVer)
	v.Set("client", c.ClientName)
	v.Set("clientver", strconv.Itoa(int(c.ClientVersion)))
	v.Set("nat", "1")
	v.Set("comp", "1")
	resp, err := c.request(ctx, "AUTH", v)
	if err != nil {
		return "", fmt.Errorf("udpapi Auth: %s", err)
	}
	switch resp.Code {
	case 201:
		// TODO Handle new anidb UDP API version available
		fallthrough
	case 200:
		parts := strings.SplitN(resp.Header, " ", 3)
		if len(parts) < 3 {
			return "", fmt.Errorf("udpapi Auth: invalid response header %q", resp.Header)
		}
		c.sessionKey.set(parts[0])
		return parts[1], nil
	default:
		return "", fmt.Errorf("udpapi Auth: bad code %d %q", resp.Code, resp.Header)
	}
}

// Logout calls the LOGOUT command.
func (c *Client) Logout(ctx context.Context) error {
	v, err := c.sessionValues()
	if err != nil {
		return fmt.Errorf("udpapi Logout: %s", err)
	}
	resp, err := c.request(ctx, "LOGOUT", v)
	if err != nil {
		return fmt.Errorf("udpapi Logout: %s", err)
	}
	c.m.SetBlock(nil)
	c.sessionKey.set("")
	switch resp.Code {
	case 203:
		return nil
	default:
		return fmt.Errorf("udpapi Logout: bad code %d %q", resp.Code, resp.Header)
	}
}

func (c *Client) FileByHash(ctx context.Context, size int64, hash string) (File, error) {
	// See https://wiki.anidb.net/UDP_API_Definition#FILE:_Retrieve_File_Data for more information
	// _, aid, eid, gid, ___, state | size, ed2k, md5, sha1, crc, ___
	// | quality, source, audio codec list, audio bitrate list, video codec, video bitrate, video resolution, extension
	// | ________ | ________
	fmask := FileFmask{0b0111_0001, 0b1111_1000, 0b1111_1111, 0b0000_0000, 0b0000_0000}
	// __, year, type, ____ | romaji name, _, english name, _, ____
	// | ep no, ep name, ep romaji name, __, ____ | group name, ___, ____
	amask := FileAmask{0b0011_0000, 0b1010_0000, 0b1110_0000, 0b1000_0000}

	data, err := c.fileByHash(ctx, size, hash, fmask, amask)
	if err != nil {
		return File{}, err
	}

	if len(data) != 26 {
		return File{}, fmt.Errorf("expected 26 fields, got %d, raw: %v", len(data), data)
	}

	var file File
	var n uint64
	var i int64

	// File ID is always present.
	n, err = strconv.ParseUint(data[0], 10, 32)
	if err != nil {
		return File{}, fmt.Errorf("invalid file ID %q: %w", data[0], err)
	}
	file.FileID = uint32(n)

	// FMASK data
	n, err = strconv.ParseUint(data[1], 10, 32)
	if err != nil {
		return File{}, fmt.Errorf("invalid anime ID %q: %w", data[1], err)
	}
	file.AnimeID = uint32(n)

	n, err = strconv.ParseUint(data[2], 10, 32)
	if err != nil {
		return File{}, fmt.Errorf("invalid episode ID %q: %w", data[2], err)
	}
	file.EpisodeID = uint32(n)

	n, err = strconv.ParseUint(data[3], 10, 32)
	if err != nil {
		return File{}, fmt.Errorf("invalid group ID %q: %w", data[3], err)
	}
	file.GroupID = uint32(n)

	n, err = strconv.ParseUint(data[4], 10, 16)
	if err != nil {
		return File{}, fmt.Errorf("invalid state %q: %w", data[4], err)
	}
	file.State = uint16(n)

	i, err = strconv.ParseInt(data[5], 10, 64)
	if err != nil {
		return File{}, fmt.Errorf("invalid size %q: %w", data[5], err)
	}
	file.Size = int(i)

	file.Ed2K = data[6]
	file.MD5 = data[7]
	file.SHA1 = data[8]
	file.CRC = data[9]
	file.Quality = data[10]
	file.Source = data[11]
	file.AudioCodec = data[12]

	n, err = strconv.ParseUint(data[13], 10, 32)
	if err != nil {
		return File{}, fmt.Errorf("invalid audio bitrate %q: %w", data[13], err)
	}
	file.AudioBitrate = uint32(n)

	file.VideoCodec = data[14]

	n, err = strconv.ParseUint(data[15], 10, 32)
	if err != nil {
		return File{}, fmt.Errorf("invalid video bitrate %q: %w", data[15], err)
	}
	file.VideoBitrate = uint32(n)

	file.VideoResolution = data[16]
	file.Extension = data[17]

	// AMASK data
	file.Year = data[18]
	file.Type = data[19]
	file.RomajiName = data[20]
	file.EnglishName = data[21]
	file.EpNum = data[22]
	file.EpName = data[23]
	file.EpRomajiName = data[24]
	file.GroupName = data[25]

	return file, nil
}

// FileByHash calls the FILE command by size+ed2k hash.
// The returned error wraps a [codes.ReturnCode] if applicable.
func (c *Client) fileByHash(ctx context.Context, size int64, hash string, fmask FileFmask, amask FileAmask) ([]string, error) {
	v, err := c.sessionValues()
	if err != nil {
		return nil, fmt.Errorf("udpapi FileByHash: %s", err)
	}
	v.Set("size", fmt.Sprintf("%d", size))
	v.Set("ed2k", hash)
	v.Set("fmask", formatMask(fmask[:]))
	v.Set("amask", formatMask(amask[:]))
	resp, err := c.request(ctx, "FILE", v)
	if err != nil {
		return nil, fmt.Errorf("udpapi FileByHash: %s", err)
	}
	if resp.Code != 220 {
		return nil, fmt.Errorf("udpapi FileByHash: got bad return code %w", resp.Code)
	}
	if n := len(resp.Rows); n != 1 {
		return nil, fmt.Errorf("udpapi FileByHash: got unexpected number of rows %d", n)
	}
	return resp.Rows[0], nil
}

// Ping calls the PING command with nat=1 and returns the port.
func (c *Client) Ping(ctx context.Context) (port string, _ error) {
	v := make(url.Values)
	v.Set("nat", "1")
	resp, err := c.request(ctx, "PING", v)
	if err != nil {
		return "", fmt.Errorf("udpapi Ping: %s", err)
	}
	if resp.Code != 300 {
		return "", fmt.Errorf("udpapi Ping: got bad return code %s", resp.Code)
	}
	if n := len(resp.Rows); n != 1 {
		return "", fmt.Errorf("udpapi Ping: got unexpected number of rows %d", n)
	}
	if n := len(resp.Rows[0]); n != 1 {
		return "", fmt.Errorf("udpapi Ping: got unexpected number of fields %d", n)
	}
	return resp.Rows[0][0], nil
}

// Uptime calls the UPTIME command and returns server uptime in milliseconds.
func (c *Client) Uptime(ctx context.Context) (uptime int, _ error) {
	v, err := c.sessionValues()
	if err != nil {
		return 0, fmt.Errorf("udpapi Uptime: %s", err)
	}
	resp, err := c.request(ctx, "UPTIME", v)
	if err != nil {
		return 0, fmt.Errorf("udpapi Uptime: %s", err)
	}
	if resp.Code != 208 {
		return 0, fmt.Errorf("udpapi Uptime: got bad return code %s", resp.Code)
	}
	if n := len(resp.Rows); n != 1 {
		return 0, fmt.Errorf("udpapi Uptime: got unexpected number of rows %d", n)
	}
	if n := len(resp.Rows[0]); n != 1 {
		return 0, fmt.Errorf("udpapi Uptime: got unexpected number of fields %d", n)
	}
	time, err := strconv.Atoi(resp.Rows[0][0])
	if err != nil {
		return 0, fmt.Errorf("udpapi Uptime: %s", err)
	}
	return time, nil
}

// request sends a request to the underlying mux, with rate limiting.
func (c *Client) request(ctx context.Context, cmd string, args url.Values) (Response, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return Response{}, err
	}
	return c.m.Request(ctx, cmd, args)
}

// sessionValues returns the values to use for the current session.
func (c *Client) sessionValues() (url.Values, error) {
	v := make(url.Values)
	key := c.sessionKey.get()
	if key == "" {
		return nil, errors.New("no session key (log in with AUTH first)")
	}
	v.Set("s", key)
	return v, nil
}
