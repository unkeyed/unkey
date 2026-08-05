#!/usr/bin/env python3

import hashlib
import http.server
import json
import os
import re
import socket
import subprocess
import sys
import threading
import time
from dataclasses import asdict, dataclass
from pathlib import Path
from urllib.parse import urlparse


LISTEN_PORT = int(sys.argv[1])
PUBLIC_HOSTNAME = urlparse(sys.argv[2]).hostname
STATIC_SOURCE_HOSTNAME = "local-api-production-local.unkey.local"
DYNAMIC_PORT_START = 20_000
DYNAMIC_PORT_COUNT = 10_000
POLL_SECONDS = 5
HOSTNAME_PATTERN = re.compile(r"^[a-zA-Z0-9.-]+$")
THREAD_HOST_PATTERN = re.compile(r"^t-([0-9a-f-]{36})-p[0-9]+\.(.+)$")
PORTAL_NAME_PATTERN = re.compile(r"[^a-zA-Z0-9-]+")

if PUBLIC_HOSTNAME is None:
    raise ValueError("PUBLIC_URL must contain a hostname")

thread_host_match = THREAD_HOST_PATTERN.fullmatch(PUBLIC_HOSTNAME)
if thread_host_match is None:
    raise ValueError("PUBLIC_URL must contain an Amp thread portal hostname")

THREAD_ID = "T-" + thread_host_match.group(1)
PORTAL_DOMAIN = thread_host_match.group(2)
AMP = os.path.expanduser("~/.amp/bin/amp")
MISE = os.path.expanduser("~/.local/bin/mise")
PORTALS_DIR = Path(".amp/portals")


@dataclass(frozen=True)
class EnvironmentRoute:
    environment_id: str
    source_hostname: str
    project_slug: str
    app_slug: str
    environment_slug: str
    workspace_slug: str


@dataclass(frozen=True)
class ActivePortal:
    environment_id: str
    source_hostname: str
    name: str
    port: int
    url: str
    title: str


STATE_LOCK = threading.Lock()
ACTIVE_PORTALS: dict[str, ActivePortal] = {}
PORTAL_PROCESSES: dict[str, subprocess.Popen[bytes]] = {}
LAST_ERROR = ""


def run_kubectl(*args: str) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [MISE, "exec", "--", "kubectl", *args],
        check=True,
        capture_output=True,
        text=True,
        timeout=30,
    )


def list_environment_routes() -> list[EnvironmentRoute]:
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
        return []

    query = """
SELECT
  route.environment_id,
  route.fully_qualified_domain_name,
  project.slug,
  app.slug,
  environment.slug,
  workspace.slug
FROM frontline_routes AS route
JOIN environments AS environment ON environment.id = route.environment_id
JOIN apps AS app ON app.id = route.app_id
JOIN projects AS project ON project.id = route.project_id
JOIN workspaces AS workspace ON workspace.id = project.workspace_id
WHERE route.sticky = 'environment'
  AND route.fully_qualified_domain_name LIKE '%.unkey.local'
ORDER BY route.environment_id, route.created_at;
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

    routes: list[EnvironmentRoute] = []
    seen_environments: set[str] = set()
    for line in result.stdout.splitlines():
        fields = line.split("\t")
        if len(fields) != 6 or fields[0] in seen_environments:
            continue
        route = EnvironmentRoute(*fields)
        if not HOSTNAME_PATTERN.fullmatch(route.source_hostname):
            continue
        routes.append(route)
        seen_environments.add(route.environment_id)
    return routes


def portal_name(environment_id: str) -> str:
    suffix = PORTAL_NAME_PATTERN.sub("-", environment_id).strip("-").lower()
    return "deployment-env-" + suffix


def portal_title(route: EnvironmentRoute) -> str:
    app = "" if route.app_slug == "default" else f"/{route.app_slug}"
    return f"{route.project_slug}{app} · {route.environment_slug}"


def allocate_port(environment_id: str, used_ports: set[int]) -> int:
    digest = hashlib.sha256(environment_id.encode()).digest()
    offset = int.from_bytes(digest[:4], "big") % DYNAMIC_PORT_COUNT
    for step in range(DYNAMIC_PORT_COUNT):
        port = DYNAMIC_PORT_START + (offset + step) % DYNAMIC_PORT_COUNT
        if port in used_ports:
            continue
        try:
            with socket.socket() as candidate:
                candidate.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
                candidate.bind(("127.0.0.1", port))
            return port
        except OSError:
            continue
    raise RuntimeError("no deployment portal ports are available")


def amp_environment() -> dict[str, str]:
    return os.environ | {
        "AMP_THREAD_ID": THREAD_ID,
        "AMP_PORTAL_DOMAIN": PORTAL_DOMAIN,
    }


def stop_process(process: subprocess.Popen[bytes]) -> None:
    if process.poll() is not None:
        return
    process.terminate()
    try:
        process.wait(timeout=5)
    except subprocess.TimeoutExpired:
        process.kill()
        process.wait(timeout=5)


def wait_until_listening(port: int, process: subprocess.Popen[bytes]) -> None:
    deadline = time.monotonic() + 10
    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise RuntimeError(f"deployment portal process exited with {process.returncode}")
        try:
            with socket.create_connection(("127.0.0.1", port), timeout=0.2):
                return
        except OSError:
            time.sleep(0.1)
    raise RuntimeError(f"deployment portal did not listen on port {port}")


def start_portal(
    route: EnvironmentRoute, port: int
) -> tuple[ActivePortal, subprocess.Popen[bytes]]:
    name = portal_name(route.environment_id)
    title = portal_title(route)
    public_url = f"https://t-{THREAD_ID[2:].lower()}-p{port}.{PORTAL_DOMAIN}/"
    process = subprocess.Popen(
        [
            sys.executable,
            "-u",
            ".amp/frontline-portal.py",
            str(port),
            public_url,
            route.source_hostname,
        ]
    )

    try:
        wait_until_listening(port, process)
        result = subprocess.run(
            [
                AMP,
                "orb",
                "portal",
                str(port),
                "--thread",
                THREAD_ID,
                "--domain",
                PORTAL_DOMAIN,
                "--name",
                name,
                "--title",
                title,
                "--description",
                f"{route.workspace_slug} deployment routed through Frontline.",
            ],
            check=True,
            capture_output=True,
            text=True,
            timeout=30,
            env=amp_environment(),
        )
        urls = [
            line.strip()
            for line in result.stdout.splitlines()
            if line.startswith("https://")
        ]
        if not urls:
            raise RuntimeError(f"Amp did not return a portal URL for {name}")
    except Exception:
        stop_process(process)
        (PORTALS_DIR / f"{name}.json").unlink(missing_ok=True)
        raise

    portal = ActivePortal(
        environment_id=route.environment_id,
        source_hostname=route.source_hostname,
        name=name,
        port=port,
        url=urls[-1],
        title=title,
    )
    print(f"Deployment portal ready: {title} -> {portal.url}", flush=True)
    return portal, process


def stop_portal(portal: ActivePortal, process: subprocess.Popen[bytes]) -> None:
    stop_process(process)
    (PORTALS_DIR / f"{portal.name}.json").unlink(missing_ok=True)


def clean_stale_manifests() -> None:
    PORTALS_DIR.mkdir(parents=True, exist_ok=True)
    for manifest in PORTALS_DIR.glob("deployment-env-*.json"):
        manifest.unlink()


def reconcile(routes: list[EnvironmentRoute]) -> None:
    dynamic_routes = {
        route.environment_id: route
        for route in routes
        if route.source_hostname != STATIC_SOURCE_HOSTNAME
    }

    with STATE_LOCK:
        current_portals = list(ACTIVE_PORTALS.items())
    for environment_id, portal in current_portals:
        with STATE_LOCK:
            process = PORTAL_PROCESSES[environment_id]
        route = dynamic_routes.get(environment_id)
        if (
            route is not None
            and route.source_hostname == portal.source_hostname
            and process.poll() is None
        ):
            continue

        stop_portal(portal, process)
        with STATE_LOCK:
            ACTIVE_PORTALS.pop(environment_id, None)
            PORTAL_PROCESSES.pop(environment_id, None)

    with STATE_LOCK:
        used_ports = {portal.port for portal in ACTIVE_PORTALS.values()}
        active_environment_ids = set(ACTIVE_PORTALS)
    for environment_id, route in sorted(dynamic_routes.items()):
        if environment_id in active_environment_ids:
            continue
        port = allocate_port(environment_id, used_ports)
        portal, process = start_portal(route, port)
        used_ports.add(port)
        with STATE_LOCK:
            ACTIVE_PORTALS[environment_id] = portal
            PORTAL_PROCESSES[environment_id] = process


def watch_routes() -> None:
    global LAST_ERROR

    cleaned = False
    while True:
        try:
            if not cleaned:
                clean_stale_manifests()
                cleaned = True
            reconcile(list_environment_routes())
            with STATE_LOCK:
                LAST_ERROR = ""
        except (OSError, RuntimeError, subprocess.SubprocessError) as error:
            message = str(error)
            with STATE_LOCK:
                if message != LAST_ERROR:
                    print(f"Unable to reconcile deployment portals: {message}", flush=True)
                LAST_ERROR = message
        time.sleep(POLL_SECONDS)


class StatusHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self) -> None:
        with STATE_LOCK:
            portals = [asdict(portal) for portal in ACTIVE_PORTALS.values()]
            error = LAST_ERROR
        body = json.dumps(
            {"status": "ok", "portals": portals, "error": error},
            indent=2,
            sort_keys=True,
        ).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, format: str, *args: object) -> None:
        return


threading.Thread(target=watch_routes, daemon=True).start()
http.server.ThreadingHTTPServer(("0.0.0.0", LISTEN_PORT), StatusHandler).serve_forever()
