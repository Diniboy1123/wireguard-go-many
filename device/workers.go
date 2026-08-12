/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2025 WireGuard LLC. All Rights Reserved.
 */

package device

import (
	"os"
	"runtime"
	"strconv"
	"sync"
)

// Upstream spawns NumCPU*3 crypto workers per Device (O(N*CPU) goroutines).
// This fork uses one process-wide pool instead. Encryption/decryption elements
// carry elem.peer/keypair so workers need no Device reference; QueueHandshakeElement
// gains a device field for the one case that didn't. Shared queues are never
// closed -- they live for the process, not any one Device's lifecycle.
var (
	sharedWorkersOnce sync.Once
	sharedEncryption  chan *QueueOutboundElementsContainer
	sharedDecryption  chan *QueueInboundElementsContainer
	sharedHandshake   chan QueueHandshakeElement
)

// sharedWorkerCount returns workers per type (default: GOMAXPROCS). Override with WG_WORKERS.
func sharedWorkerCount() int {
	if s := os.Getenv("WG_WORKERS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return n
		}
	}
	return runtime.GOMAXPROCS(0)
}

// ensureSharedWorkers lazily starts the shared crypto pool (sync.Once; safe from every NewDevice).
func ensureSharedWorkers() {
	sharedWorkersOnce.Do(func() {
		sharedEncryption = make(chan *QueueOutboundElementsContainer, QueueOutboundSize)
		sharedDecryption = make(chan *QueueInboundElementsContainer, QueueInboundSize)
		sharedHandshake = make(chan QueueHandshakeElement, QueueHandshakeSize)

		n := sharedWorkerCount()
		for i := 0; i < n; i++ {
			go RoutineEncryption(i + 1)
			go RoutineDecryption(i + 1)
			go RoutineHandshake(i + 1)
		}
	})
}
