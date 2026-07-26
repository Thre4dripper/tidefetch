package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Thre4dripper/tidefetch/pkg/aria2"
)

// --- messages ---------------------------------------------------------------

type tickMsg time.Time

// pollMsg carries a full refresh of daemon state.
type pollMsg struct {
	stat    aria2.GlobalStat
	active  []aria2.Status
	waiting []aria2.Status
	stopped []aria2.Status
	err     error
}

// notifMsg wraps an asynchronous aria2 event.
type notifMsg struct {
	n  aria2.Notification
	ok bool // false when the notification channel closed (connection lost)
}

// reconMsg is the outcome of a reconnect attempt.
type reconMsg struct {
	client  *aria2.Client
	version string
	err     error
}

// actionMsg is the outcome of a fire-and-forget RPC action.
type actionMsg struct {
	toast string
	level toastLevel
	err   error
}

// addedMsg is the outcome of submitting the Add form.
type addedMsg struct {
	count int
	err   error
}

// detailMsg carries per-download detail data.
type detailMsg struct {
	gid     string
	status  aria2.Status
	peers   []aria2.Peer
	servers []aria2.ServerGroup
	options aria2.Options
	err     error
}

// globalOptsMsg carries daemon-wide options for the settings view.
type globalOptsMsg struct {
	opts aria2.Options
	err  error
}

// historySavedMsg signals async history persistence finished.
type historySavedMsg struct{ err error }

// --- commands ---------------------------------------------------------------

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func rpcCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 4*time.Second)
}

// pollCmd fetches global stat + all three download lists.
func pollCmd(c *aria2.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := rpcCtx()
		defer cancel()
		var m pollMsg
		var err error
		if m.stat, err = c.GetGlobalStat(ctx); err != nil {
			m.err = err
			return m
		}
		if m.active, err = c.TellActive(ctx); err != nil {
			m.err = err
			return m
		}
		if m.waiting, err = c.TellWaiting(ctx, 0, 1000); err != nil {
			m.err = err
			return m
		}
		if m.stopped, err = c.TellStopped(ctx, 0, 1000); err != nil {
			m.err = err
			return m
		}
		return m
	}
}

// listenNotifs waits for the next aria2 event.
func listenNotifs(ch <-chan aria2.Notification) tea.Cmd {
	return func() tea.Msg {
		n, ok := <-ch
		return notifMsg{n: n, ok: ok}
	}
}

// doRPC runs fn and reports a toast (or error) when done.
func doRPC(toast string, fn func(ctx context.Context) error) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := rpcCtx()
		defer cancel()
		if err := fn(ctx); err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{toast: toast, level: toastOK}
	}
}

// detailCmd fetches everything the details view needs.
func detailCmd(c *aria2.Client, gid string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := rpcCtx()
		defer cancel()
		m := detailMsg{gid: gid}
		var err error
		if m.status, err = c.TellStatus(ctx, gid); err != nil {
			m.err = err
			return m
		}
		m.options, _ = c.GetOption(ctx, gid)
		if m.status.IsTorrent() && m.status.Status == "active" {
			m.peers, _ = c.GetPeers(ctx, gid)
		}
		if m.status.Status == "active" {
			m.servers, _ = c.GetServers(ctx, gid)
		}
		return m
	}
}
