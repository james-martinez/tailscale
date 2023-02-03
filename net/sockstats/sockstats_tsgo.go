// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

//go:build tailscale_go && (unix || windows)

package sockstats

import (
	"context"
	"net"
	"sync"
	"sync/atomic"

	"tailscale.com/net/interfaces"
	"tailscale.com/wgengine/monitor"
)

type sockStatCounters struct {
	txBytes, rxBytes                           atomic.Uint64
	rxByBytesByInterface, txByBytesByInterface map[string]*atomic.Uint64
}

var (
	mu               sync.Mutex
	countersByLabel  map[string]*sockStatCounters = make(map[string]*sockStatCounters)
	currentInterface string
	knownInterfaces  map[string]int = make(map[string]int)
	usedInterfaces   map[string]int = make(map[string]int)
)

func withSockStats(ctx context.Context, label string) context.Context {
	mu.Lock()
	defer mu.Unlock()
	counters, ok := countersByLabel[label]
	if !ok {
		counters = &sockStatCounters{
			rxByBytesByInterface: make(map[string]*atomic.Uint64),
			txByBytesByInterface: make(map[string]*atomic.Uint64),
		}
		for iface := range knownInterfaces {
			counters.rxByBytesByInterface[iface] = &atomic.Uint64{}
			counters.txByBytesByInterface[iface] = &atomic.Uint64{}
		}
		countersByLabel[label] = counters
	}

	didRead := func(n int) {
		counters.rxBytes.Add(uint64(n))
		if currentInterface != "" {
			if a := counters.rxByBytesByInterface[currentInterface]; a != nil {
				a.Add(uint64(n))
			}
		}
	}
	didWrite := func(n int) {
		counters.txBytes.Add(uint64(n))
		if currentInterface != "" {
			if a := counters.txByBytesByInterface[currentInterface]; a != nil {
				a.Add(uint64(n))
			}
		}
	}

	return net.WithSockTrace(ctx, &net.SockTrace{
		DidRead:  didRead,
		DidWrite: didWrite,
	})
}

func get() *SockStats {
	mu.Lock()
	defer mu.Unlock()

	sockStats := &SockStats{
		Stats:      make(map[string]SockStat),
		Interfaces: make([]string, 0, len(usedInterfaces)),
	}
	for iface := range usedInterfaces {
		sockStats.Interfaces = append(sockStats.Interfaces, iface)
	}

	for label, counters := range countersByLabel {
		sockStats.Stats[label] = SockStat{
			TxBytes:              int64(counters.txBytes.Load()),
			RxBytes:              int64(counters.rxBytes.Load()),
			TxByBytesByInterface: make(map[string]int64),
			RxByBytesByInterface: make(map[string]int64),
		}
		for iface, a := range counters.rxByBytesByInterface {
			sockStats.Stats[label].RxByBytesByInterface[iface] = int64(a.Load())
		}
		for iface, a := range counters.txByBytesByInterface {
			sockStats.Stats[label].TxByBytesByInterface[iface] = int64(a.Load())
		}
	}

	return sockStats
}

func setLinkMonitor(lm *monitor.Mon) {
	mu.Lock()
	defer mu.Unlock()

	// We intentionally populate all known interfaces now, so that we can
	// increment stats for them without holding mu.
	state := lm.InterfaceState()
	for iface := range state.Interface {
		knownInterfaces[iface] = 1
	}
	if iface := state.DefaultRouteInterface; iface != "" {
		currentInterface = iface
		usedInterfaces[iface] = 1
	}

	lm.RegisterChangeCallback(func(changed bool, state *interfaces.State) {
		if changed {
			if iface := state.DefaultRouteInterface; iface != "" {
				mu.Lock()
				defer mu.Unlock()
				// Ignore changes to unknown interfaces -- it would require
				// updating the tx/rxByBytesByInterface maps and thus
				// additional locking for every read/write. Most of the time
				// the set of interfaces is static.
				if _, ok := knownInterfaces[iface]; ok {
					currentInterface = iface
					usedInterfaces[iface] = 1
				} else {
					currentInterface = ""
				}
			}
		}
	})
}
