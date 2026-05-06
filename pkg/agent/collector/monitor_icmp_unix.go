//go:build !windows

package collector

import (
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

func pingHost(target string, count, timeoutSec int) (*pingStats, error) {
	pinger, err := probing.NewPinger(target)
	if err != nil {
		return nil, err
	}

	pinger.Count = count
	pinger.Timeout = time.Duration(timeoutSec) * time.Second
	pinger.Interval = 100 * time.Millisecond
	pinger.SetPrivileged(false)

	if err := pinger.Run(); err != nil {
		// 非特权 UDP ICMP 不可用时回退到 raw socket（需要 root / CAP_NET_RAW）
		pinger.SetPrivileged(true)
		if err := pinger.Run(); err != nil {
			return nil, err
		}
	}

	s := pinger.Statistics()
	return &pingStats{
		PacketsSent: s.PacketsSent,
		PacketsRecv: s.PacketsRecv,
		AvgRttMs:    s.AvgRtt.Milliseconds(),
		PacketLoss:  s.PacketLoss,
	}, nil
}
