// Package peer carries cross-region traffic between frontlines over a
// backhaul of long-lived, multiplexed HTTP/2 connections.
//
// The entry frontline routes, evaluates policies, logs, and picks the target
// instance. It sends the request as an HTTP/2 stream on a pooled connection
// to a frontline in the instance's region. That frontline acts as a relay: it
// verifies the signed target, checks that the target is a live local pod of
// the named deployment, dials it, and streams bytes. It never routes or
// forwards again.
//
// Plain requests are ordinary streams. Protocol upgrades travel as classic
// HTTP/2 CONNECT streams that carry the upgrade token and request target in
// headers, so a new WebSocket costs one round trip on an open connection.
//
// For a plain request, the relay answers with a private 1xx status once its
// dial to the pod succeeds. Until then nothing has reached the pod, so the
// entry can retry another instance; [ReplayBody] keeps the request body
// replayable for that window. An upgrade has no body, so it gets no ack and a
// failed upgrade may move on to the next instance.
//
// New streams on an idle connection are limited by TCP slow start, so the
// [Pool] keeps many warm connections per region rather than a few busy ones.
package peer
