// Package buffer provides a generic buffered channel implementation with configurable capacity and drop behavior.
//
// The Buffer type encapsulates a channel and offers a simple interface to add items and
// consume them safely. It supports buffering with optional drop-on-full behavior, which
// is particularly useful for high-throughput logging, metrics collection, and other
// scenarios where dropping newer items is preferable to blocking producers.
package buffer
