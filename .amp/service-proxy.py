#!/usr/bin/env python3

import select
import socket
import socketserver
import sys


LISTEN_PORT = int(sys.argv[1])
UPSTREAM_PORT = int(sys.argv[2])
BUFFER_SIZE = 64 * 1024


class ProxyHandler(socketserver.BaseRequestHandler):
    def handle(self) -> None:
        try:
            upstream = socket.create_connection(("127.0.0.1", UPSTREAM_PORT), timeout=5)
        except OSError:
            return

        with upstream:
            upstream.settimeout(None)
            sockets = (self.request, upstream)
            while True:
                try:
                    readable, _, _ = select.select(sockets, (), ())
                except OSError:
                    return

                for source in readable:
                    try:
                        data = source.recv(BUFFER_SIZE)
                    except OSError:
                        return

                    if not data:
                        return

                    target = upstream if source is self.request else self.request
                    try:
                        target.sendall(data)
                    except OSError:
                        return


class ProxyServer(socketserver.ThreadingTCPServer):
    allow_reuse_address = True
    daemon_threads = True


ProxyServer(("0.0.0.0", LISTEN_PORT), ProxyHandler).serve_forever()
