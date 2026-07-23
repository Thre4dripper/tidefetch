package aria2

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// RPCError is a JSON-RPC error object returned by aria2.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string { return fmt.Sprintf("aria2 rpc error %d: %s", e.Code, e.Message) }

// ErrClosed is returned for calls made after the connection is gone.
var ErrClosed = errors.New("aria2: connection closed")

type request struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type response struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *RPCError       `json:"error"`
	Method string          `json:"method"` // set for notifications
	Params json.RawMessage `json:"params"`
}

// Client is a thread-safe aria2 JSON-RPC client over WebSocket.
type Client struct {
	url    string
	secret string

	conn    *websocket.Conn
	writeMu sync.Mutex

	pendingMu sync.Mutex
	pending   map[string]chan *response

	nextID atomic.Uint64

	notifs chan Notification

	closeOnce sync.Once
	done      chan struct{}
	errMu     sync.Mutex
	err       error
}

// Dial connects to an aria2 RPC endpoint, e.g. "ws://127.0.0.1:6800/jsonrpc".
// secret may be empty when the daemon runs without --rpc-secret.
func Dial(ctx context.Context, url, secret string) (*Client, error) {
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, resp, err := dialer.DialContext(ctx, url, http.Header{})
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("aria2: dial %s: %w", url, err)
	}
	c := &Client{
		url:     url,
		secret:  secret,
		conn:    conn,
		pending: make(map[string]chan *response),
		notifs:  make(chan Notification, 64),
		done:    make(chan struct{}),
	}
	go c.readLoop()
	return c, nil
}

// URL returns the endpoint this client is connected to.
func (c *Client) URL() string { return c.url }

// Notifications returns the channel of asynchronous aria2 events.
// The channel is closed when the connection dies.
func (c *Client) Notifications() <-chan Notification { return c.notifs }

// Done is closed when the connection is torn down.
func (c *Client) Done() <-chan struct{} { return c.done }

// Err reports why the connection closed, if it did.
func (c *Client) Err() error {
	c.errMu.Lock()
	defer c.errMu.Unlock()
	return c.err
}

// Close tears down the connection.
func (c *Client) Close() error {
	c.shutdown(ErrClosed)
	return nil
}

func (c *Client) shutdown(err error) {
	c.closeOnce.Do(func() {
		c.errMu.Lock()
		c.err = err
		c.errMu.Unlock()
		close(c.done)
		c.conn.Close()
		c.pendingMu.Lock()
		for id, ch := range c.pending {
			close(ch)
			delete(c.pending, id)
		}
		c.pendingMu.Unlock()
		close(c.notifs)
	})
}

func (c *Client) readLoop() {
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.shutdown(err)
			return
		}
		var resp response
		if err := json.Unmarshal(data, &resp); err != nil {
			continue
		}
		if resp.Method != "" { // notification
			for _, gid := range parseNotificationGIDs(resp.Params) {
				select {
				case c.notifs <- Notification{Method: resp.Method, GID: gid}:
				default: // never block the read loop
				}
			}
			continue
		}
		c.pendingMu.Lock()
		ch, ok := c.pending[resp.ID]
		if ok {
			delete(c.pending, resp.ID)
		}
		c.pendingMu.Unlock()
		if ok {
			ch <- &resp
			close(ch)
		}
	}
}

// Call invokes an arbitrary RPC method. The secret token is prepended
// automatically for aria2.* methods. result may be nil.
func (c *Client) Call(ctx context.Context, method string, result any, params ...any) error {
	select {
	case <-c.done:
		return ErrClosed
	default:
	}

	full := params
	if c.secret != "" && len(method) > 6 && method[:6] == "aria2." {
		full = append([]any{"token:" + c.secret}, params...)
	}
	if full == nil {
		full = []any{}
	}

	id := fmt.Sprintf("%d", c.nextID.Add(1))
	req := request{JSONRPC: "2.0", ID: id, Method: method, Params: full}

	ch := make(chan *response, 1)
	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	c.writeMu.Lock()
	err := c.conn.WriteJSON(req)
	c.writeMu.Unlock()
	if err != nil {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return fmt.Errorf("aria2: write: %w", err)
	}

	select {
	case <-ctx.Done():
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
		return ctx.Err()
	case <-c.done:
		return ErrClosed
	case resp, ok := <-ch:
		if !ok || resp == nil {
			return ErrClosed
		}
		if resp.Error != nil {
			return resp.Error
		}
		if result != nil {
			if err := json.Unmarshal(resp.Result, result); err != nil {
				return fmt.Errorf("aria2: decode result of %s: %w", method, err)
			}
		}
		return nil
	}
}

// ---------------------------------------------------------------------------
// Adding downloads
// ---------------------------------------------------------------------------

// AddURI queues a download from one or more mirror URIs (HTTP/HTTPS/FTP/SFTP/magnet).
func (c *Client) AddURI(ctx context.Context, uris []string, opts Options) (string, error) {
	var gid string
	if opts == nil {
		opts = Options{}
	}
	err := c.Call(ctx, "aria2.addUri", &gid, uris, opts)
	return gid, err
}

// AddTorrent uploads raw .torrent file bytes. webSeeds may be nil.
func (c *Client) AddTorrent(ctx context.Context, torrent []byte, webSeeds []string, opts Options) (string, error) {
	var gid string
	if opts == nil {
		opts = Options{}
	}
	if webSeeds == nil {
		webSeeds = []string{}
	}
	err := c.Call(ctx, "aria2.addTorrent", &gid, base64.StdEncoding.EncodeToString(torrent), webSeeds, opts)
	return gid, err
}

// AddTorrentFile reads a .torrent file from disk and queues it.
func (c *Client) AddTorrentFile(ctx context.Context, path string, opts Options) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return c.AddTorrent(ctx, data, nil, opts)
}

// AddMetalink uploads raw .metalink/.meta4 file bytes. Returns the new GIDs.
func (c *Client) AddMetalink(ctx context.Context, metalink []byte, opts Options) ([]string, error) {
	var gids []string
	if opts == nil {
		opts = Options{}
	}
	err := c.Call(ctx, "aria2.addMetalink", &gids, base64.StdEncoding.EncodeToString(metalink), opts)
	return gids, err
}

// AddMetalinkFile reads a metalink file from disk and queues it.
func (c *Client) AddMetalinkFile(ctx context.Context, path string, opts Options) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return c.AddMetalink(ctx, data, opts)
}

// ---------------------------------------------------------------------------
// Controlling downloads
// ---------------------------------------------------------------------------

// Pause pauses a download.
func (c *Client) Pause(ctx context.Context, gid string) error {
	return c.Call(ctx, "aria2.pause", nil, gid)
}

// ForcePause pauses without contacting trackers/servers.
func (c *Client) ForcePause(ctx context.Context, gid string) error {
	return c.Call(ctx, "aria2.forcePause", nil, gid)
}

// PauseAll pauses every download.
func (c *Client) PauseAll(ctx context.Context) error {
	return c.Call(ctx, "aria2.pauseAll", nil)
}

// Unpause resumes a paused download.
func (c *Client) Unpause(ctx context.Context, gid string) error {
	return c.Call(ctx, "aria2.unpause", nil, gid)
}

// UnpauseAll resumes every paused download.
func (c *Client) UnpauseAll(ctx context.Context) error {
	return c.Call(ctx, "aria2.unpauseAll", nil)
}

// Remove removes an active/waiting download (moves it to stopped).
func (c *Client) Remove(ctx context.Context, gid string) error {
	return c.Call(ctx, "aria2.remove", nil, gid)
}

// ForceRemove removes without any cleanup steps.
func (c *Client) ForceRemove(ctx context.Context, gid string) error {
	return c.Call(ctx, "aria2.forceRemove", nil, gid)
}

// RemoveDownloadResult erases a stopped download from aria2's memory.
func (c *Client) RemoveDownloadResult(ctx context.Context, gid string) error {
	return c.Call(ctx, "aria2.removeDownloadResult", nil, gid)
}

// PurgeDownloadResult erases all completed/errored/removed downloads.
func (c *Client) PurgeDownloadResult(ctx context.Context) error {
	return c.Call(ctx, "aria2.purgeDownloadResult", nil)
}

// ChangePosition moves a waiting download in the queue.
// how is one of "POS_SET", "POS_CUR", "POS_END".
func (c *Client) ChangePosition(ctx context.Context, gid string, pos int, how string) (int, error) {
	var newPos int
	err := c.Call(ctx, "aria2.changePosition", &newPos, gid, pos, how)
	return newPos, err
}

// ---------------------------------------------------------------------------
// Querying
// ---------------------------------------------------------------------------

// TellStatus fetches full status for one download. keys narrows returned fields (nil = all).
func (c *Client) TellStatus(ctx context.Context, gid string, keys ...string) (Status, error) {
	var st Status
	var err error
	if len(keys) > 0 {
		err = c.Call(ctx, "aria2.tellStatus", &st, gid, keys)
	} else {
		err = c.Call(ctx, "aria2.tellStatus", &st, gid)
	}
	return st, err
}

// TellActive lists downloads currently in progress.
func (c *Client) TellActive(ctx context.Context) ([]Status, error) {
	var list []Status
	err := c.Call(ctx, "aria2.tellActive", &list)
	return list, err
}

// TellWaiting lists queued/paused downloads.
func (c *Client) TellWaiting(ctx context.Context, offset, num int) ([]Status, error) {
	var list []Status
	err := c.Call(ctx, "aria2.tellWaiting", &list, offset, num)
	return list, err
}

// TellStopped lists finished/errored/removed downloads.
func (c *Client) TellStopped(ctx context.Context, offset, num int) ([]Status, error) {
	var list []Status
	err := c.Call(ctx, "aria2.tellStopped", &list, offset, num)
	return list, err
}

// GetURIs lists the URIs of a download.
func (c *Client) GetURIs(ctx context.Context, gid string) ([]URI, error) {
	var list []URI
	err := c.Call(ctx, "aria2.getUris", &list, gid)
	return list, err
}

// GetFiles lists the files of a download.
func (c *Client) GetFiles(ctx context.Context, gid string) ([]File, error) {
	var list []File
	err := c.Call(ctx, "aria2.getFiles", &list, gid)
	return list, err
}

// GetPeers lists BitTorrent peers of a download.
func (c *Client) GetPeers(ctx context.Context, gid string) ([]Peer, error) {
	var list []Peer
	err := c.Call(ctx, "aria2.getPeers", &list, gid)
	return list, err
}

// GetServers lists currently connected HTTP/FTP servers of a download.
func (c *Client) GetServers(ctx context.Context, gid string) ([]ServerGroup, error) {
	var list []ServerGroup
	err := c.Call(ctx, "aria2.getServers", &list, gid)
	return list, err
}

// GetGlobalStat returns overall transfer statistics.
func (c *Client) GetGlobalStat(ctx context.Context) (GlobalStat, error) {
	var st GlobalStat
	err := c.Call(ctx, "aria2.getGlobalStat", &st)
	return st, err
}

// GetVersion returns the aria2 version and enabled features.
func (c *Client) GetVersion(ctx context.Context) (VersionInfo, error) {
	var v VersionInfo
	err := c.Call(ctx, "aria2.getVersion", &v)
	return v, err
}

// GetSessionInfo returns the aria2 session id.
func (c *Client) GetSessionInfo(ctx context.Context) (SessionInfo, error) {
	var s SessionInfo
	err := c.Call(ctx, "aria2.getSessionInfo", &s)
	return s, err
}

// ---------------------------------------------------------------------------
// Options
// ---------------------------------------------------------------------------

// GetOption returns per-download options.
func (c *Client) GetOption(ctx context.Context, gid string) (Options, error) {
	var o Options
	err := c.Call(ctx, "aria2.getOption", &o, gid)
	return o, err
}

// ChangeOption updates per-download options.
func (c *Client) ChangeOption(ctx context.Context, gid string, opts Options) error {
	return c.Call(ctx, "aria2.changeOption", nil, gid, opts)
}

// GetGlobalOption returns daemon-wide options.
func (c *Client) GetGlobalOption(ctx context.Context) (Options, error) {
	var o Options
	err := c.Call(ctx, "aria2.getGlobalOption", &o)
	return o, err
}

// ChangeGlobalOption updates daemon-wide options.
func (c *Client) ChangeGlobalOption(ctx context.Context, opts Options) error {
	return c.Call(ctx, "aria2.changeGlobalOption", nil, opts)
}

// ---------------------------------------------------------------------------
// Session control
// ---------------------------------------------------------------------------

// SaveSession persists the current session to the --save-session file.
func (c *Client) SaveSession(ctx context.Context) error {
	return c.Call(ctx, "aria2.saveSession", nil)
}

// Shutdown asks the daemon to exit gracefully.
func (c *Client) Shutdown(ctx context.Context) error {
	return c.Call(ctx, "aria2.shutdown", nil)
}

// ForceShutdown asks the daemon to exit immediately.
func (c *Client) ForceShutdown(ctx context.Context) error {
	return c.Call(ctx, "aria2.forceShutdown", nil)
}

// ---------------------------------------------------------------------------
// system.multicall
// ---------------------------------------------------------------------------

// MulticallCall describes one method invocation inside a batch.
type MulticallCall struct {
	Method string `json:"methodName"`
	Params []any  `json:"params"`
}

// Multicall executes several methods in one round trip. Each result element
// is the raw JSON result for the corresponding call; per-call errors are
// returned in errs (nil when the call succeeded).
func (c *Client) Multicall(ctx context.Context, calls []MulticallCall) (results []json.RawMessage, errs []error, err error) {
	prepared := make([]MulticallCall, len(calls))
	for i, call := range calls {
		p := call.Params
		if c.secret != "" && len(call.Method) > 6 && call.Method[:6] == "aria2." {
			p = append([]any{"token:" + c.secret}, p...)
		}
		if p == nil {
			p = []any{}
		}
		prepared[i] = MulticallCall{Method: call.Method, Params: p}
	}
	var raw []json.RawMessage
	if err := c.Call(ctx, "system.multicall", &raw, prepared); err != nil {
		return nil, nil, err
	}
	results = make([]json.RawMessage, len(raw))
	errs = make([]error, len(raw))
	for i, item := range raw {
		var asArray []json.RawMessage
		if e := json.Unmarshal(item, &asArray); e == nil && len(asArray) == 1 {
			results[i] = asArray[0]
			continue
		}
		var rpcErr RPCError
		if e := json.Unmarshal(item, &rpcErr); e == nil && rpcErr.Message != "" {
			errs[i] = &rpcErr
			continue
		}
		results[i] = item
	}
	return results, errs, nil
}
