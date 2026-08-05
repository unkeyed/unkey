#!/usr/bin/env python3

import hashlib
import os
import re
import select
import socket
import socketserver
import ssl
import subprocess
import sys
import threading
import time
from urllib.parse import urlparse


LISTEN_PORT = int(sys.argv[1])
PUBLIC_HOSTNAME = urlparse(sys.argv[2]).hostname
SOURCE_HOSTNAME = sys.argv[3]
UPSTREAM_PORT = 9443
BUFFER_SIZE = 64 * 1024
HEALTH_PATH = b"/_unkey/internal/health/ready"
HOSTNAME_PATTERN = re.compile(r"^[a-zA-Z0-9.-]+$")
ROUTE_READY = threading.Event()

if PUBLIC_HOSTNAME is None or not HOSTNAME_PATTERN.fullmatch(PUBLIC_HOSTNAME):
    raise ValueError("PUBLIC_URL must contain a valid hostname")
if not HOSTNAME_PATTERN.fullmatch(SOURCE_HOSTNAME):
    raise ValueError("source route must be a valid hostname")

MISE = os.path.expanduser("~/.local/bin/mise")
ROUTE_ID = "flr_orb_" + hashlib.sha256(PUBLIC_HOSTNAME.encode()).hexdigest()[:24]


def run_kubectl(*args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [MISE, "exec", "--", "kubectl", *args],
        check=True,
        capture_output=True,
        text=True,
        timeout=30,
    )


def register_route() -> bool:
    pod = run_kubectl(
        "-n",
        "unkey",
        "get",
        "pod",
        "-l",
        "app=mysql",
        "-o",
        "jsonpath={.items[0].metadata.name}",
    ).stdout.strip()
    if not pod:
        return False

    query = f"""
INSERT INTO frontline_routes (
  id,
  project_id,
  app_id,
  deployment_id,
  environment_id,
  fully_qualified_domain_name,
  sticky,
  created_at,
  updated_at
)
SELECT
  '{ROUTE_ID}',
  project_id,
  app_id,
  deployment_id,
  environment_id,
  '{PUBLIC_HOSTNAME}',
  sticky,
  CAST(UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3)) * 1000 AS SIGNED),
  NULL
FROM frontline_routes
WHERE fully_qualified_domain_name = '{SOURCE_HOSTNAME}'
ON DUPLICATE KEY UPDATE
  project_id = VALUES(project_id),
  app_id = VALUES(app_id),
  deployment_id = VALUES(deployment_id),
  environment_id = VALUES(environment_id),
  sticky = VALUES(sticky),
  updated_at = CAST(UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3)) * 1000 AS SIGNED);

SELECT COUNT(*)
FROM frontline_routes AS portal
JOIN frontline_routes AS source
  ON source.fully_qualified_domain_name = '{SOURCE_HOSTNAME}'
  AND portal.environment_id = source.environment_id
WHERE portal.fully_qualified_domain_name = '{PUBLIC_HOSTNAME}';
"""
    result = run_kubectl(
        "-n",
        "unkey",
        "exec",
        pod,
        "--",
        "env",
        "MYSQL_PWD=password",
        "mysql",
        "-N",
        "-u",
        "unkey",
        "unkey",
        "-e",
        query,
    )
    return result.stdout.strip().endswith("1")


def register_route_when_available() -> None:
    while True:
        try:
            ready = register_route()
        except (OSError, subprocess.SubprocessError):
            ready = False

        if ready:
            ROUTE_READY.set()
            print(
                f"Frontline route ready: https://{PUBLIC_HOSTNAME} -> {SOURCE_HOSTNAME}",
                flush=True,
            )
            return

        time.sleep(5)


# Amp terminates public TLS. Re-encrypt over loopback so Frontline observes the
# original HTTPS scheme; its development certificate is intentionally self-signed.
TLS_CONTEXT = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
TLS_CONTEXT.check_hostname = False
TLS_CONTEXT.verify_mode = ssl.CERT_NONE


def read_headers(client: socket.socket) -> bytes:
    request = bytearray()
    while b"\r\n\r\n" not in request and len(request) < BUFFER_SIZE:
        data = client.recv(BUFFER_SIZE - len(request))
        if not data:
            break
        request.extend(data)
    return bytes(request)


def request_path(request: bytes) -> bytes:
    request_line = request.partition(b"\r\n")[0].split()
    if len(request_line) < 2:
        return b""
    return request_line[1].partition(b"?")[0]


class ProxyHandler(socketserver.BaseRequestHandler):
    def handle(self) -> None:
        try:
            initial_request = read_headers(self.request)
        except OSError:
            return

        if not initial_request:
            return

        if not ROUTE_READY.is_set() and request_path(initial_request) != HEALTH_PATH:
            body = f"Waiting for deployment route {SOURCE_HOSTNAME}\n".encode()
            self.request.sendall(
                b"HTTP/1.1 503 Service Unavailable\r\n"
                b"Content-Type: text/plain; charset=utf-8\r\n"
                + f"Content-Length: {len(body)}\r\n".encode()
                + b"Connection: close\r\n\r\n"
                + body
            )
            return

        try:
            upstream_socket = socket.create_connection(
                ("127.0.0.1", UPSTREAM_PORT), timeout=5
            )
            upstream = TLS_CONTEXT.wrap_socket(
                upstream_socket, server_hostname=SOURCE_HOSTNAME
            )
            upstream.sendall(initial_request)
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


threading.Thread(target=register_route_when_available, daemon=True).start()
ProxyServer(("0.0.0.0", LISTEN_PORT), ProxyHandler).serve_forever()
