import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import time
import unittest


ROOT = Path(__file__).resolve().parents[1]


class LocalDevTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.temporary = tempfile.TemporaryDirectory(prefix="unkey-local-dev-")
        cls.addClassCleanup(cls.temporary.cleanup)
        cls.root = Path(cls.temporary.name)
        cls.dev = cls.root / "dev"
        cls.dev.mkdir()
        for name in ("Tiltfile", "Dockerfile.binary", "04-seed-workspace.sql", "clickhouse-local-admin.sql", "init-clickhouse.sh", "start-cluster.sh", "stripe-webhook-secret.sh"):
            shutil.copyfile(ROOT / "dev" / name, cls.dev / name)
        (cls.dev / "k8s").symlink_to(ROOT / "dev/k8s", target_is_directory=True)
        (cls.root / "pkg").symlink_to(ROOT / "pkg", target_is_directory=True)
        (cls.root / ".mise").symlink_to(ROOT / ".mise", target_is_directory=True)
        cls.dashboard_env = cls.root / "web/apps/dashboard/.env"
        cls.dashboard_env.parent.mkdir(parents=True)
        cls.bin = cls.root / "mock-bin"
        cls.bin.mkdir()
        cls.log = cls.root / "commands.jsonl"
        stub = cls.bin / "stub"
        stub.write_text(
            f"#!{sys.executable}\n"
            "import json, os, pathlib, sys, time\n"
            "name = pathlib.Path(sys.argv[0]).name\n"
            "args = sys.argv[1:]\n"
            "with open(os.environ['COMMAND_LOG'], 'a') as log:\n"
            "    log.write(json.dumps([name, *args]) + '\\n')\n"
            "if os.environ.get('FAIL_COMMAND') in (name, ' '.join([name, *args])):\n"
            "    sys.exit(17)\n"
            "if name == 'mise':\n"
            "    if args == ['env', '--json']: print(json.dumps({'PATH': os.environ.get('MISE_TEST_PATH', os.environ['PATH']), 'GOTOOLCHAIN': 'local'}))\n"
            "    elif args == ['which', 'kubectl']: print(pathlib.Path(sys.argv[0]).parent / 'kubectl')\n"
            "    else: sys.exit(18)\n"
            "elif name == 'sudo':\n"
            "    if args == ['-v']: sys.exit(0)\n"
            "    if args == ['-n', 'true']: sys.exit(1)\n"
            "    os.execvpe(args[0], args, dict(os.environ, PATH='/usr/bin:/bin'))\n"
            "elif name == 'go':\n"
            "    if args == ['env', 'GOARCH']: print('amd64')\n"
            "    else: sys.exit(1)\n"
            "elif name == 'stripe':\n"
            "    if args == ['config', '--list']: print('account_id = stale')\n"
            "    elif args == ['listen', '--print-secret', '--skip-update']:\n"
            "        if os.environ.get('STRIPE_TEST_PID_FILE'): pathlib.Path(os.environ['STRIPE_TEST_PID_FILE']).write_text(str(os.getpid()))\n"
            "        time.sleep(float(os.environ.get('STRIPE_TEST_DELAY', '0')))\n"
            "        print(os.environ.get('STRIPE_TEST_SECRET', ''))\n"
            "        sys.exit(int(os.environ.get('STRIPE_TEST_EXIT', '0')))\n"
            "    else: sys.exit(18)\n"
            "elif name == 'kubectl' and args[:5] == ['-n', 'kube-system', 'patch', 'configmap', 'cilium-config']:\n"
            "    print(os.environ.get('CONFIG_VERSION', '41'), end='')\n"
            "elif name == 'minikube' and args[:2] == ['profile', 'list']:\n"
            "    print(os.environ.get('PROFILES', '{\"valid\": [], \"invalid\": []}'))\n"
            "elif name == 'minikube' and args[:1] == ['status'] and os.environ.get('STOPPED') == '1':\n"
            "    sys.exit(7)\n"
            "elif name == 'ctlptl' and args[:1] == ['get']:\n"
            "    print(os.environ.get('CLUSTER', '{\"registry\": \"ctlptl-registry\", \"status\": {\"localRegistryHosting\": {\"host\": \"localhost:5000\"}}}'))\n"
        )
        stub.chmod(0o755)
        for name in ("mise", "sudo", "go", "stripe", "kubectl", "ctlptl", "minikube", "tilt", "docker", "pnpm"):
            (cls.bin / name).symlink_to(stub)
        cls.env = dict(os.environ, PATH=f"{cls.bin}:{os.environ['PATH']}", COMMAND_LOG=str(cls.log), MINIKUBE_HOME=str(cls.root / "minikube-home"))
        tilt = shutil.which("tilt")
        if tilt is None:
            raise RuntimeError("Run with mise exec -- python3 dev/local_dev_test.py")

        def evaluate(*args, **env):
            result = subprocess.run(
                [tilt, "alpha", "tiltfile-result", "-f", str(cls.dev / "Tiltfile"), "--", *args],
                cwd=cls.root, env=dict(cls.env, **env), capture_output=True, text=True, timeout=120,
            )
            if result.returncode:
                raise RuntimeError(result.stderr)
            data = json.loads(result.stdout)
            if data["Error"]:
                raise RuntimeError(data["Error"])
            manifests = {manifest["Name"]: manifest for manifest in data["Manifests"]}
            return manifests, set(data["EnabledManifests"])

        cls.evaluate = staticmethod(evaluate)
        cls.default, _ = evaluate()
        cls.stripe, _ = evaluate(STRIPE_TEST_SECRET="whsec_fixture")
        cls.orb, _ = evaluate("--orb")
        _, cls.api_enabled = evaluate("api")
        _, cls.krane_enabled = evaluate("krane")
        _, cls.dashboard_enabled = evaluate("dashboard")
        (cls.dev / ".env.github").write_text("UNKEY_GITHUB_APP_ID=123\n")
        (cls.dev / ".github-private-key.pem").write_text("fixture\n")
        cls.github, _ = evaluate()

    def setUp(self):
        self.log.write_text("")

    def command(self, args, cwd=None, **env):
        return subprocess.run(
            args, cwd=cwd or self.dev, env=dict(self.env, **env),
            capture_output=True, text=True, timeout=10,
        )

    def calls(self):
        return [json.loads(line) for line in self.log.read_text().splitlines()]

    def update_command(self, manifests, name):
        return manifests[name]["DeployTarget"]["UpdateCmdSpec"]["args"]

    def test_direct_tilt_uses_mise_tools_during_evaluation_and_updates(self):
        with tempfile.TemporaryDirectory(dir=self.root) as directory:
            for name in ("stripe", "kubectl", "go", "helm"):
                executable = Path(directory) / name
                executable.write_text("#!/bin/sh\necho 'wrong tool from inherited PATH' >&2\nexit 41\n")
                executable.chmod(0o755)
            env = {
                "PATH": f"{directory}:{self.env['PATH']}",
                "MISE_TEST_PATH": self.env["PATH"],
                "STRIPE_TEST_SECRET": "whsec_" + "fixture",
            }
            manifests, _ = self.evaluate(**env)
            self.assertIn("stripe-listen-dashboard", manifests)
            result = subprocess.run(
                [shutil.which("tilt"), "ci", "--port", "0", "--timeout", "30s",
                 "-f", str(self.dev / "Tiltfile"), "--", "namespace"],
                cwd=self.root, env=dict(self.env, **env), capture_output=True, text=True, timeout=60,
            )
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertIn(["kubectl", "apply", "-f", "k8s/manifests/namespace.yaml"], self.calls())

    def test_tunnel_uses_pinned_kubectl_when_sudo_resets_path(self):
        result = self.command(["bash", str(ROOT / ".mise/tasks/tunnel")], FAIL_COMMAND="kubectl")
        self.assertEqual(result.returncode, 17, result.stderr)
        self.assertIn(["sudo", str(self.bin / "kubectl"), "port-forward", "-n", "frontline", "svc/frontline", "443:443", "80:80"], self.calls())
        self.assertIn(["kubectl", "port-forward", "-n", "frontline", "svc/frontline", "443:443", "80:80"], self.calls())

    def test_selected_service_includes_dependencies_without_unrelated_services(self):
        self.assertEqual(self.api_enabled, {
            "api", "api-compile", "mysql", "clickhouse", "redis",
            "namespace", "github-credentials", "uncategorized", "storage-provisioner",
        })
        self.assertTrue({
            "krane", "ctrl-api", "cilium-ready", "cilium-policies",
            "topolvm-vg", "topolvm-storageclass", "rbac", "node-labels",
        } <= self.krane_enabled)
        self.assertTrue({"dashboard", "hubble", "otel-collector", "logdrain"}.isdisjoint(self.krane_enabled))

    def test_dashboard_builds_sdk_before_start_and_watches_only_inputs(self):
        self.assertIn("api-sdk", self.dashboard_enabled)
        self.assertNotIn("api-sdk", self.api_enabled)
        sdk = self.default["api-sdk"]["DeployTarget"]
        self.assertEqual(set(sdk["Deps"]), {
            str(self.root / "web/internal/api" / path)
            for path in ("src", "tsconfig.json", "package.json")
        })
        self.assertTrue(sdk["AllowParallel"])
        self.assertIn("api-sdk", self.default["dashboard"]["ResourceDependencies"])
        self.assertEqual(self.default["dashboard"]["DeployTarget"]["Deps"], [])
        command = self.update_command(self.default, "api-sdk")
        result = self.command(command)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.calls(), [["pnpm", "--dir=../web", "--filter=@unkey/api", "build"]])
        result = self.command(command, FAIL_COMMAND="pnpm")
        self.assertEqual(result.returncode, 17)

    def test_dashboard_readiness_does_not_render_a_page(self):
        for manifests in (self.default, self.orb, self.stripe):
            self.assertEqual(manifests["dashboard"]["DeployTarget"]["ReadinessProbe"], {
                "tcpSocket": {"port": 3000},
                "periodSeconds": 1,
                "failureThreshold": 150,
            })

    def test_stripe_is_optional_without_a_secret(self):
        self.assertNotIn("stripe-webhook-secret", self.default)
        self.assertNotIn("stripe-listen-dashboard", self.default)
        self.assertNotIn("stripe-listen-ctrl", self.default)
        self.assertEqual(self.default["stripe-credentials"]["ResourceDependencies"], ["namespace"])
        for name in ("ctrl-api", "ctrl-worker"):
            self.assertIn("stripe-credentials", self.default[name]["ResourceDependencies"])

    def test_configured_stripe_preserves_secret_before_consumer_order(self):
        for name in ("dashboard", "stripe-credentials"):
            self.assertIn("stripe-webhook-secret", self.stripe[name]["ResourceDependencies"])
        for listener, consumer in (("stripe-listen-dashboard", "dashboard"), ("stripe-listen-ctrl", "ctrl-api")):
            self.assertEqual(self.stripe[listener]["ResourceDependencies"], [consumer])

    def test_missing_stripe_cli_is_optional(self):
        result = self.command(
            [shutil.which("bash"), str(ROOT / "dev/stripe-webhook-secret.sh")],
            PATH=str(self.root / "missing-bin"),
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout, "")

    def test_stripe_check_is_bounded_and_cleans_up(self):
        with tempfile.TemporaryDirectory() as temporary:
            pid_file = self.root / "stripe.pid"
            for delay in ("0", "10"):
                with self.subTest(delay=delay):
                    start = time.monotonic()
                    result = self.command(
                        ["bash", str(ROOT / "dev/stripe-webhook-secret.sh")],
                        TMPDIR=temporary,
                        STRIPE_TEST_PID_FILE=str(pid_file),
                        STRIPE_TEST_DELAY=delay,
                        STRIPE_TEST_SECRET="whsec_fixture",
                    )
                    elapsed = time.monotonic() - start
                    self.assertEqual(result.returncode, 0, result.stderr)
                    self.assertEqual(result.stdout, "whsec_fixture" if delay == "0" else "")
                    self.assertLess(elapsed, 3 if delay == "0" else 8)
                    if delay == "10":
                        self.assertGreaterEqual(elapsed, 4.5)
                    with self.assertRaises(ProcessLookupError):
                        os.kill(int(pid_file.read_text()), 0)
                    self.assertEqual(list(Path(temporary).iterdir()), [])

    def test_failed_stripe_auth_or_invalid_secret_does_not_overwrite_env(self):
        for env in (
            {"FAIL_COMMAND": "stripe"},
            {"STRIPE_TEST_SECRET": "whsec_fixture", "STRIPE_TEST_EXIT": "17"},
            {"STRIPE_TEST_SECRET": ""},
            {"STRIPE_TEST_SECRET": "not-a-secret"},
            {"STRIPE_TEST_SECRET": "whsec_bad&secret"},
            {"STRIPE_TEST_SECRET": "whsec_fixture", "STRIPE_TEST_DELAY": "10"},
        ):
            with self.subTest(env=env):
                for path in (self.dev / ".env.stripe", self.dashboard_env):
                    path.write_text("STRIPE_WEBHOOK_SECRET=whsec_existing\nOTHER=keep\n")
                manifests, _ = self.evaluate(**env)
                self.assertNotIn("stripe-webhook-secret", manifests)
                self.assertNotIn("stripe-listen-dashboard", manifests)
                self.assertNotIn("stripe-listen-ctrl", manifests)
                self.assertEqual(manifests["stripe-credentials"]["ResourceDependencies"], ["namespace"])
                self.assertNotIn("stripe-webhook-secret", manifests["dashboard"]["ResourceDependencies"])
                for path in (self.dev / ".env.stripe", self.dashboard_env):
                    self.assertEqual(path.read_text(), "STRIPE_WEBHOOK_SECRET=whsec_existing\nOTHER=keep\n")

    def test_secret_update_is_shared_and_repeatable(self):
        (self.dev / ".env.stripe").write_text("STRIPE_WEBHOOK_SECRET=whsec_old\nOTHER=keep\n")
        self.dashboard_env.write_text("OTHER=dashboard\n")
        spec = self.stripe["stripe-webhook-secret"]["DeployTarget"]["UpdateCmdSpec"]
        self.assertEqual(spec["env"], ["STRIPE_WEBHOOK_SECRET=whsec_fixture"])
        for _ in range(2):
            result = self.command(self.update_command(self.stripe, "stripe-webhook-secret"), STRIPE_WEBHOOK_SECRET="whsec_fixture")
            self.assertEqual(result.returncode, 0, result.stderr)
        self.assertFalse(any(call[0] == "stripe" for call in self.calls()))
        self.assertEqual((self.dev / ".env.stripe").read_text(), "STRIPE_WEBHOOK_SECRET=whsec_fixture\nOTHER=keep\n")
        self.assertEqual(self.dashboard_env.read_text(), "OTHER=dashboard\nSTRIPE_WEBHOOK_SECRET=whsec_fixture\n")

    def test_network_storage_and_service_dependencies_remain_required(self):
        required = {
            "cilium-policies": {"namespace", "cilium-ready"},
            "krane": {"cilium-policies", "node-labels", "rbac", "ctrl-api", "registry", "topolvm-storageclass", "krane-compile"},
            "topolvm-storageclass": {"topolvm-controller", "topolvm-node", "topolvm-lvmd-0"},
            "ctrl-api": {"mysql", "clickhouse", "restate", "vault", "rbac", "control-api-compile"},
            "ctrl-worker": {"mysql", "clickhouse", "restate", "registry", "rbac", "vault", "control-worker-compile"},
            "rbac": {"namespace"},
            "prometheus": {"namespace"},
            "otel-collector": {"namespace"},
        }
        for name, dependencies in required.items():
            with self.subTest(resource=name):
                self.assertTrue(dependencies <= set(self.default[name]["ResourceDependencies"]))
        for name in ("topolvm-controller", "topolvm-node", "topolvm-lvmd-0"):
            self.assertIn("topolvm-vg", self.default[name]["ResourceDependencies"])
            self.assertIn("uncategorized", self.default[name]["ResourceDependencies"])
        for name in ("mysql", "clickhouse", "redis", "s3", "restate"):
            self.assertIn("namespace", self.default[name]["ResourceDependencies"])
        for name in ("mysql", "clickhouse", "s3", "restate"):
            self.assertIn("storage-provisioner", self.default[name]["ResourceDependencies"])
        for name in ("cilium-ready", "hubble", "topolvm-vg"):
            self.assertTrue(self.default[name]["DeployTarget"]["AllowParallel"])
        self.assertIn("heimdall", self.default)
        self.assertNotIn("cilium-network-policy-crd", self.default)
        self.assertNotIn("cilium-ready", self.orb)
        self.assertIn("cilium-network-policy-crd", self.orb["krane"]["ResourceDependencies"])

    def test_storage_provisioner_replaces_addon_only_after_successful_disable(self):
        command = self.update_command(self.default, "storage-provisioner")
        result = self.command(command)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.calls(), [
            ["minikube", "addons", "disable", "storage-provisioner"],
            ["kubectl", "apply", "-f", "k8s/manifests/storage-provisioner.yaml"],
        ])
        self.log.write_text("")
        result = self.command(command, FAIL_COMMAND="minikube")
        self.assertEqual(result.returncode, 17)
        self.assertEqual(self.calls(), [["minikube", "addons", "disable", "storage-provisioner"]])
        manifest = (ROOT / "dev/k8s/manifests/storage-provisioner.yaml").read_text()
        self.assertIn("hostNetwork: true", manifest)
        self.assertIn('name: KUBERNETES_SERVICE_HOST\n          value: "127.0.0.1"', manifest)
        self.assertIn('name: KUBERNETES_SERVICE_PORT\n          value: "8443"', manifest)
        self.assertIn("serviceAccountName: storage-provisioner", manifest)
        self.assertIn("path: /tmp\n        type: Directory", manifest)
        self.assertNotIn("addonmanager.kubernetes.io/mode", manifest)
        self.assertNotIn("kind: PersistentVolume", manifest)

    def test_local_updates_do_not_hold_a_global_lock(self):
        for manifests in (self.default, self.orb, self.github, self.stripe):
            for name, manifest in manifests.items():
                target = manifest["DeployTarget"]
                if "UpdateCmdSpec" in target:
                    with self.subTest(resource=name):
                        self.assertTrue(target["AllowParallel"])
        self.assertEqual(self.default["seed"]["ResourceDependencies"], ["mysql", "seed-compile"])
        self.assertFalse(self.default["seed-compile"]["ResourceDependencies"])
        self.assertIn("go build", " ".join(self.update_command(self.default, "seed-compile")))
        self.assertNotIn("go run", " ".join(self.update_command(self.default, "seed")))

    def test_namespaced_objects_deploy_with_their_consumers(self):
        for name in ("api", "ctrl-api", "ctrl-worker", "vault", "frontline", "krane", "logdrain", "s3", "prometheus", "vector-logs"):
            with self.subTest(resource=name):
                documents = self.default[name]["DeployTarget"]["yaml"].split("\n---\n")
                self.assertTrue(any(
                    "kind: ConfigMap\n" in document and f"\n  name: {name}-config\n" in document
                    for document in documents
                ))
        for name in ("s3", "restate"):
            self.assertIn("kind: PersistentVolumeClaim", self.default[name]["DeployTarget"]["yaml"])
        uncategorized = self.default["uncategorized"]["DeployTarget"]["yaml"]
        for document in uncategorized.split("\n---\n"):
            if any(f"kind: {kind}" in document.splitlines() for kind in ("ConfigMap", "PersistentVolumeClaim", "ServiceAccount")):
                self.assertNotIn("\n  namespace: unkey\n", document)
                self.assertNotIn("\n  namespace: frontline\n", document)

    def test_database_images_are_pulled_without_local_builds(self):
        for name, schema_path, image in (
            ("mysql", "/schema/unkey", "vitess/vttestserver:v24.0.3-mysql80@sha256:"),
            ("clickhouse", "/opt/clickhouse-schemas", "clickhouse/clickhouse-server:26.2.1.1139@sha256:"),
        ):
            with self.subTest(database=name):
                manifest = self.default[name]
                self.assertEqual(manifest["ImageTargets"], [])
                yaml = manifest["DeployTarget"]["yaml"]
                self.assertIn(image, yaml)
                self.assertIn(f"mountPath: {schema_path}", yaml)
                self.assertIn(f"name: {name}-schema", yaml)
                self.assertIn("kind: ConfigMap", yaml)
        mysql_yaml = self.default["mysql"]["DeployTarget"]["yaml"]
        self.assertIn("zzz-seed.sql:", mysql_yaml)
        self.assertNotIn("USE unkey;", mysql_yaml)
        self.assertIn("--persistent_mode", mysql_yaml)
        self.assertIn("--data_dir=/vt/vtdataroot", mysql_yaml)
        readiness = self.default["clickhouse"]["DeployTarget"]["yaml"].split("readinessProbe:")[1].split("resources:")[0]
        self.assertIn("httpGet:", readiness)
        self.assertIn("path: /ping", readiness)
        self.assertNotIn("initialDelaySeconds:", readiness)

    def test_database_input_changes_roll_pods_without_image_builds(self):
        paths = [self.dev / "04-seed-workspace.sql", self.dev / "init-clickhouse.sh"]
        originals = [path.read_text() for path in paths]
        try:
            for path, original in zip(paths, originals):
                path.write_text(original + "\n")
            changed, _ = self.evaluate()
        finally:
            for path, original in zip(paths, originals):
                path.write_text(original)
        for name in ("mysql", "clickhouse"):
            with self.subTest(database=name):
                old_yaml = self.default[name]["DeployTarget"]["yaml"]
                new_yaml = changed[name]["DeployTarget"]["yaml"]
                self.assertIn("      annotations:\n        checksum/init:", new_yaml)
                old_checksum = next(line for line in old_yaml.splitlines() if "checksum/init:" in line)
                new_checksum = next(line for line in new_yaml.splitlines() if "checksum/init:" in line)
                self.assertNotEqual(old_checksum, new_checksum)
                self.assertEqual(changed[name]["ImageTargets"], [])

    def test_every_cron_waits_for_the_worker(self):
        crons = {
            name: manifest for name, manifest in self.default.items()
            if "kind: CronJob\n" in manifest["DeployTarget"].get("yaml", "")
        }
        self.assertIn("restate-deploy-spend-check", crons)
        self.assertIn("restate-clickhouse-user-reconcile", crons)
        for name, manifest in crons.items():
            with self.subTest(resource=name):
                self.assertEqual(manifest["ResourceDependencies"], ["restate", "ctrl-worker"])

    def test_worker_forward_reaches_its_listening_port(self):
        forwards = self.default["ctrl-worker"]["DeployTarget"]["portForwardTemplateSpec"]["forwards"]
        self.assertEqual([(f["localPort"], f["containerPort"]) for f in forwards], [(7092, 9080)])

    def test_cilium_readiness_fails_closed(self):
        command = self.update_command(self.default, "cilium-ready")
        result = self.command(command)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.calls(), [
            ["kubectl", "-n", "kube-system", "rollout", "status", "daemonset/cilium", "--timeout=180s"],
            ["kubectl", "wait", "--for=condition=Established", "crd/ciliumnetworkpolicies.cilium.io", "crd/ciliumclusterwidenetworkpolicies.cilium.io", "--timeout=60s"],
            ["kubectl", "-n", "kube-system", "rollout", "status", "deployment/coredns", "--timeout=180s"],
        ])
        self.log.write_text("")
        result = self.command(command, FAIL_COMMAND="kubectl")
        self.assertEqual(result.returncode, 17)
        self.assertEqual(len(self.calls()), 1)

    def test_missing_github_credentials_use_zero_without_relaxing_auth(self):
        def configmap(manifests, name):
            return next(
                document
                for manifest in manifests.values()
                for document in manifest["DeployTarget"].get("yaml", "").split("\n---\n")
                if "kind: ConfigMap\n" in document and f"\n  name: {name}-config\n" in document
            )

        for name in ("api", "ctrl-api", "ctrl-worker"):
            yaml = configmap(self.default, name)
            self.assertIn("app_id = 0", yaml)
            self.assertNotIn("app_id = ${UNKEY_GITHUB_APP_ID}", yaml)
            self.assertIn("app_id = ${UNKEY_GITHUB_APP_ID}", configmap(self.github, name))
        self.assertNotIn("allow_unauthenticated_deployments = true", configmap(self.default, "ctrl-api"))

    def test_hubble_rollout_changes_only_with_config_version(self):
        patches = []
        for version in ("41", "41", "42"):
            self.log.write_text("")
            result = self.command(["bash", str(ROOT / "dev/setup-hubble.sh")], CONFIG_VERSION=version)
            self.assertEqual(result.returncode, 0, result.stderr)
            calls = self.calls()
            self.assertEqual(len(calls), 4)
            self.assertEqual(calls[1][:3], ["kubectl", "apply", "-f"])
            self.assertEqual(calls[2][:6], ["kubectl", "-n", "kube-system", "patch", "daemonset", "cilium"])
            patches.append(json.loads(calls[2][-1]))
            self.assertEqual(calls[3], ["kubectl", "-n", "kube-system", "rollout", "status", "daemonset", "cilium", "--timeout=120s"])
        self.assertEqual(patches[0], patches[1])
        self.assertNotEqual(patches[1], patches[2])
        self.assertEqual(patches[2]["spec"]["template"]["metadata"]["annotations"], {"unkey.dev/cilium-config-version": "42"})
        self.log.write_text("")
        result = self.command(["bash", str(ROOT / "dev/setup-hubble.sh")], FAIL_COMMAND="kubectl")
        self.assertEqual(result.returncode, 17)
        self.assertEqual(len(self.calls()), 1)

    def test_dev_applies_cluster_once_and_forwards_options(self):
        result = self.command(["bash", str(ROOT / ".mise/tasks/dev"), "api", "--topolvm_backing_size=32G"], cwd=self.root)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.calls(), [
            ["minikube", "profile", "list", "-o", "json"],
            ["ctlptl", "apply", "-f", "./dev/cluster.yaml"],
            ["tilt", "up", "-f", "./dev/Tiltfile", "--", "api", "--topolvm_backing_size=32G"],
        ])
        self.log.write_text("")
        result = self.command(["bash", str(ROOT / ".mise/tasks/dev")], cwd=self.root, FAIL_COMMAND="ctlptl")
        self.assertEqual(result.returncode, 17)
        self.assertEqual(len(self.calls()), 2)
        cluster = (ROOT / "dev/cluster.yaml").read_text()
        self.assertIn('"--wait=apiserver,kubelet,node_ready"', cluster)
        self.assertIn('"--cni=cilium"', cluster)
        self.assertIn('"--addons=metrics-server"', cluster)
        self.assertIn('"--extra-config=kubelet.serialize-image-pulls=false"', cluster)

    def test_orb_forwards_resource_selection_and_preserves_network_mode(self):
        result = self.command(["bash", str(ROOT / ".mise/tasks/dev-orb"), "api"], cwd=self.root, PORT="10350")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.calls(), [
            ["minikube", "profile", "list", "-o", "json"],
            ["ctlptl", "apply", "-f", "./dev/cluster.orb.yaml"],
            ["tilt", "up", "--file", "./dev/Tiltfile", "--host", "0.0.0.0",
             "--port", "10350", "--stream", "--", "--orb=true", "api"],
        ])
        cluster = (ROOT / "dev/cluster.orb.yaml").read_text()
        self.assertIn('"--wait=apiserver,kubelet,node_ready"', cluster)
        self.assertIn('"--cni=bridge"', cluster)
        self.assertIn('"--extra-config=kubelet.serialize-image-pulls=false"', cluster)

    def test_existing_clusters_are_never_reapplied_or_recreated(self):
        for stopped in (False, True):
            with self.subTest(stopped=stopped):
                self.log.write_text("")
                result = self.command(
                    ["bash", str(ROOT / "dev/start-cluster.sh"), "changed-cluster.yaml"],
                    PROFILES=json.dumps({"valid": [{"Name": "minikube"}], "invalid": []}),
                    STOPPED="1" if stopped else "0",
                )
                self.assertEqual(result.returncode, 0, result.stderr)
                calls = self.calls()
                self.assertNotIn(["ctlptl", "apply", "-f", "changed-cluster.yaml"], calls)
                self.assertNotIn(["minikube", "delete", "-p", "minikube"], calls)
                starts = [call for call in calls if call[:2] == ["minikube", "start"]]
                self.assertEqual(starts, [["minikube", "start", "-p", "minikube", "--wait=apiserver,kubelet,node_ready", "--delete-on-failure=false"]] if stopped else [])

    def test_cluster_inspection_errors_do_not_trigger_creation(self):
        for env in (
            {"PROFILES": "not-json"},
            {"PROFILES": '{"valid": [], "invalid": [{"Name": "minikube"}]}'},
            {"FAIL_COMMAND": "minikube"},
            {"PROFILES": '{"valid": [{"Name": "minikube"}], "invalid": []}', "CLUSTER": "{}"},
            {"PROFILES": '{"valid": [{"Name": "minikube"}], "invalid": []}', "STOPPED": "1",
             "FAIL_COMMAND": "minikube start -p minikube --wait=apiserver,kubelet,node_ready --delete-on-failure=false"},
        ):
            with self.subTest(env=env):
                self.log.write_text("")
                result = self.command(["bash", str(ROOT / "dev/start-cluster.sh"), "cluster.yaml"], **env)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(any(call[:2] == ["ctlptl", "apply"] for call in self.calls()))

    def test_unlisted_profile_directory_prevents_creation(self):
        with tempfile.TemporaryDirectory(dir=self.root) as directory:
            profile = Path(directory) / ".minikube/profiles/minikube"
            profile.mkdir(parents=True)
            for home in (directory, str(Path(directory) / ".minikube")):
                with self.subTest(home=home):
                    self.log.write_text("")
                    result = self.command(["bash", str(ROOT / "dev/start-cluster.sh"), "cluster.yaml"], MINIKUBE_HOME=home)
                    self.assertNotEqual(result.returncode, 0)
                    self.assertEqual(self.calls(), [["minikube", "profile", "list", "-o", "json"]])

    def test_down_preserves_cluster_data(self):
        result = self.command(["bash", str(ROOT / ".mise/tasks/down")])
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.calls(), [["minikube", "stop", "-p", "minikube"]])


if __name__ == "__main__":
    unittest.main()
