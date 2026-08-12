/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2025 WireGuard LLC. All Rights Reserved.
 */

package device

import (
	"runtime"
)

// outboundQueue/inboundQueue/handshakeQueue replaced by process-wide shared
// channels in device/workers.go; ref-counted close machinery no longer needed.

type autodrainingInboundQueue struct {
	c chan *QueueInboundElementsContainer
}

// newAutodrainingInboundQueue returns a channel that will be drained when it gets GC'd.
// It is useful in cases in which is it hard to manage the lifetime of the channel.
// The returned channel must not be closed. Senders should signal shutdown using
// some other means, such as sending a sentinel nil values.
func newAutodrainingInboundQueue(device *Device) *autodrainingInboundQueue {
	q := &autodrainingInboundQueue{
		c: make(chan *QueueInboundElementsContainer, QueueInboundSize),
	}
	runtime.SetFinalizer(q, device.flushInboundQueue)
	return q
}

func (device *Device) flushInboundQueue(q *autodrainingInboundQueue) {
	for {
		select {
		case elemsContainer := <-q.c:
			elemsContainer.Lock()
			for _, elem := range elemsContainer.elems {
				device.PutMessageBuffer(elem.buffer)
				device.PutInboundElement(elem)
			}
			device.PutInboundElementsContainer(elemsContainer)
		default:
			return
		}
	}
}

type autodrainingOutboundQueue struct {
	c chan *QueueOutboundElementsContainer
}

// newAutodrainingOutboundQueue returns a channel that will be drained when it gets GC'd.
// It is useful in cases in which is it hard to manage the lifetime of the channel.
// The returned channel must not be closed. Senders should signal shutdown using
// some other means, such as sending a sentinel nil values.
// All sends to the channel must be best-effort, because there may be no receivers.
func newAutodrainingOutboundQueue(device *Device) *autodrainingOutboundQueue {
	q := &autodrainingOutboundQueue{
		c: make(chan *QueueOutboundElementsContainer, QueueOutboundSize),
	}
	runtime.SetFinalizer(q, device.flushOutboundQueue)
	return q
}

func (device *Device) flushOutboundQueue(q *autodrainingOutboundQueue) {
	for {
		select {
		case elemsContainer := <-q.c:
			elemsContainer.Lock()
			for _, elem := range elemsContainer.elems {
				device.PutMessageBuffer(elem.buffer)
				device.PutOutboundElement(elem)
			}
			device.PutOutboundElementsContainer(elemsContainer)
		default:
			return
		}
	}
}
