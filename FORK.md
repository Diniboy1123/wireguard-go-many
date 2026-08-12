# This fork

Patched for running many (hundreds) of `device.Device` instances in one Go
process, each maintaining a separate WireGuard tunnel. Upstream wireguard-go
is sized for one or a handful of tunnels per process; at a few hundred, the
per-Device goroutine overhead adds up fast. Linux only -- other platforms
aren't a target here and haven't been tested.

No API changes. `device.NewDevice(tun, bind, logger)` and
`conn.NewDefaultBind()` have the same signatures as upstream; everything
below happens internally.

## What changed

**One shared crypto worker pool for the whole process** (`device/workers.go`).
Upstream spawns `NumCPU*3` goroutines per Device for encryption, decryption,
and handshake processing. With many Devices that's
`NumCPU*3*deviceCount` goroutines before counting any peers. This fork
moves those three queues and their workers to a single pool shared by every
Device, sized at `GOMAXPROCS*3` total (override with `WG_WORKERS`), started
lazily on the first `NewDevice` call. `RoutineEncryption`/
`RoutineDecryption`/`RoutineHandshake` are free functions now instead of
`*Device` methods; `QueueHandshakeElement` gained a `device` field since a
raw handshake message doesn't carry a peer reference yet the way encryption/
decryption elements do.

## What that costs

- `IsUnderLoad()`'s handshake-flood threshold is now process-wide, not
  per-Device -- a flood against one Device's peers can make another
  Device's peers start requiring cookies too. Fine for outbound-only
  tunnels to trusted peers (nobody can flood you with unsolicited
  handshakes); a real isolation loss if a Device here ever needs to accept
  inbound connections from untrusted peers.
- Per-worker verbose logging is gone (workers used to log against their
  owning Device's logger; now there's no single device to log against).
- Each Device still opens its own 2 UDP sockets + a cancellation pipe --
  fd pressure at very high Device counts is unchanged from upstream. Raise
  `ulimit -n` if running toward the high end of "hundreds."

## Tunables

| env var      | default      | notes                              |
|--------------|--------------|------------------------------------|
| `WG_WORKERS` | `GOMAXPROCS` | shared crypto workers per type     |

## Measured

500 Devices (250 client/server pairs) in one process, real handshakes, real
ChaCha20-Poly1305 traffic:

|                    | upstream | this fork |
|--------------------|----------|-----------|
| Goroutines/Device  | 35.0     | 8.05      |

Goroutine count is the important number at scale: upstream's 35/Device is
linear in Device count forever; this fork's stays flat because the crypto
workers are a fixed-size pool instead of growing with N.

Cross-checked against unmodified upstream wireguard-go in a separate
process (real handshake, real traffic, not just self-consistency): clean
interop, no protocol changes here.
