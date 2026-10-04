// Package server delivers DNS messages between sockets and a dnswire.Handler;
// it holds no answering logic. [UDP] wraps the miekg server, [TCP] is a custom
// server, and [Probe] checks either from readiness probes.
//
// TCP is custom because the miekg server (codeberg.org/miekg/dns v0.6.115)
// closes a connection as soon as its read loop ends while pipelined handlers
// may still be writing, writes replies without a lock or write deadline so
// concurrent replies can interleave, retries failed Accept calls forever, and
// ignores the Shutdown context, so it can't bound a graceful drain. [TCP]
// writes each reply whole under a deadline, closes a connection after its
// replies are written, fails Serve when the listener fails, and drains
// accepted requests until the Shutdown context expires.
package server
