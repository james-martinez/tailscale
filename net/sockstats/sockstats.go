// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

// Package sockstats collects statistics about network sockets used by
// the Tailscale client. The context where sockets are used must be
// instrumented with the WithSockStats() function.
//
// Only available on Posix platforms when built with Tailscale's fork of Go.
package sockstats

import (
	"context"

	"tailscale.com/wgengine/monitor"
)

type SockStats struct {
	Stats      map[string]SockStat
	Interfaces []string
}

type SockStat struct {
	TxBytes              int64
	RxBytes              int64
	TxByBytesByInterface map[string]int64
	RxByBytesByInterface map[string]int64
}

func WithSockStats(ctx context.Context, label string) context.Context {
	return withSockStats(ctx, label)
}

func Get() *SockStats {
	return get()
}

func SetLinkMonitor(lm *monitor.Mon) {
	setLinkMonitor(lm)
}
