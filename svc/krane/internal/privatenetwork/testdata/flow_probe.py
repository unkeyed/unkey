import json
import socket
import socketserver
import sys
import threading
import time
import uuid


class TCPHandler(socketserver.StreamRequestHandler):
    def handle(self):
        for line in self.rfile:
            self.wfile.write(line)
            self.wfile.flush()


class UDPHandler(socketserver.BaseRequestHandler):
    def handle(self):
        data, sock = self.request
        sock.sendto(data, self.client_address)


class Flow:
    def __init__(self, host, protocol):
        self.sock = socket.socket(socket.AF_INET, protocol)
        self.pending = b""
        self.sock.settimeout(0.5)
        try:
            self.sock.connect((host, 8080))
        except OSError:
            self.sock.close()
            raise

    def exchange(self):
        token = uuid.uuid4().hex.encode() + b"\n"
        deadline = time.monotonic() + 0.5
        try:
            self.sock.settimeout(0.5)
            self.sock.sendall(token)
            while time.monotonic() < deadline:
                self.sock.settimeout(max(0.001, deadline - time.monotonic()))
                data = self.sock.recv(4096)
                if not data:
                    return False
                self.pending += data
                while b"\n" in self.pending:
                    line, self.pending = self.pending.split(b"\n", 1)
                    if line + b"\n" == token:
                        return True
            return False
        except OSError:
            return False


if sys.argv[1] == "server":
    with socketserver.ThreadingUDPServer(("0.0.0.0", 8080), UDPHandler) as udp:
        threading.Thread(target=udp.serve_forever, daemon=True).start()
        with socketserver.ThreadingTCPServer(("0.0.0.0", 8080), TCPHandler) as tcp:
            tcp.serve_forever()
else:
    held = {}
    try:
        for line in sys.stdin:
            command = line.strip()
            if command not in ("fresh", "open", "held"):
                raise ValueError(command)
            result = {}
            for name, protocol in (("tcp", socket.SOCK_STREAM), ("udp", socket.SOCK_DGRAM)):
                if command == "held":
                    result[name] = held[name].exchange()
                    continue
                try:
                    flow = Flow(sys.argv[2], protocol)
                except OSError:
                    result[name] = False
                    continue
                result[name] = flow.exchange()
                if command == "open" and result[name]:
                    if name in held:
                        held[name].sock.close()
                    held[name] = flow
                else:
                    flow.sock.close()
            print(json.dumps(result), flush=True)
    finally:
        for flow in held.values():
            flow.sock.close()
