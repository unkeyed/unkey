#!/usr/bin/env python3

import http.client
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


LISTEN_PORT = int(sys.argv[1])
UPSTREAM_PORT = int(sys.argv[2])
HOP_BY_HOP_HEADERS = {
    "connection",
    "keep-alive",
    "proxy-authenticate",
    "proxy-authorization",
    "te",
    "trailers",
    "transfer-encoding",
    "upgrade",
}


class ProxyHandler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def do_GET(self) -> None:
        if self.path == "/":
            self._readiness()
            return
        self._proxy()

    do_DELETE = do_GET
    do_HEAD = do_GET
    do_OPTIONS = do_GET
    do_PATCH = do_GET
    do_POST = do_GET
    do_PUT = do_GET

    def _readiness(self) -> None:
        upstream = http.client.HTTPConnection("127.0.0.1", UPSTREAM_PORT, timeout=2)
        try:
            upstream.connect()
        except OSError:
            self._respond(503, b"upstream unavailable\n")
        else:
            upstream.close()
            self._respond(200, b"upstream ready\n")

    def _proxy(self) -> None:
        content_length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(content_length) if content_length else None
        headers = {
            name: value
            for name, value in self.headers.items()
            if name.lower() not in HOP_BY_HOP_HEADERS and name.lower() != "host"
        }
        upstream = http.client.HTTPConnection("127.0.0.1", UPSTREAM_PORT, timeout=30)

        try:
            upstream.request(self.command, self.path, body=body, headers=headers)
            response = upstream.getresponse()
            payload = response.read()
        except (OSError, http.client.HTTPException):
            self._respond(502, b"upstream request failed\n")
            return
        finally:
            upstream.close()

        self.send_response(response.status)
        for name, value in response.getheaders():
            if name.lower() not in HOP_BY_HOP_HEADERS and name.lower() != "content-length":
                self.send_header(name, value)
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        if self.command != "HEAD":
            self.wfile.write(payload)

    def _respond(self, status: int, body: bytes) -> None:
        self.send_response(status)
        self.send_header("Content-Type", "text/plain; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        if self.command != "HEAD":
            self.wfile.write(body)


ThreadingHTTPServer(("0.0.0.0", LISTEN_PORT), ProxyHandler).serve_forever()
