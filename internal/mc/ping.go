package mc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	mcnet "github.com/Tnze/go-mc/net"
)

type PingResult struct {
	Online    bool
	Players   int
	RTTTime   time.Duration
	TotalTime time.Duration
}

func Ping(ctx context.Context, addr string, timeout time.Duration) (PingResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	totalStart := time.Now() // 总时长计时器

	// resolve address
	resolved, err := ResolveAddr(ctx, addr)
	if err != nil {
		return PingResult{}, fmt.Errorf("resolve: %w", err)
	}
	// establish a TCP connection
	var d net.Dialer
	rawConn, err := d.DialContext(ctx, "tcp", resolved.DialAddr)
	if err != nil {
		return PingResult{}, fmt.Errorf("dial %s: %w", resolved.DialAddr, err)
	}
	// close Nagle
	if tcpConn, ok := rawConn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
	}
	// pack as MC connection
	conn := mcnet.WrapConn(rawConn)
	defer func() { _ = conn.Close() }()
	// set socket deadline(resolve timeout)
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.Socket.SetDeadline(deadline)
	}
	// lisen ctx cancel signal
	stop := context.AfterFunc(ctx, func() {
		_ = conn.Close()
	})
	defer stop()
	// call protocol func
	raw, RTTDuration, err := ServerListPing(conn, resolved.Host, resolved.Port)
	if err != nil {
		return PingResult{}, fmt.Errorf("server list ping: %w", err)
	}
	// decode json
	var status struct {
		Players struct {
			Online int `json:"online"`
		} `json:"players"`
	}
	if err := json.Unmarshal(raw, &status); err != nil {
		return PingResult{}, fmt.Errorf("json decode: %w", err)
	}

	return PingResult{
		Online:    true,
		Players:   status.Players.Online,
		RTTTime:   RTTDuration,
		TotalTime: time.Since(totalStart),
	}, nil
}
