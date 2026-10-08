import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
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
        for name in ("Tiltfile", "Dockerfile.binary", "Dockerfile.mysql", "Dockerfile.clickhouse", "start-cluster.sh", "stripe-webhook-secret.mjs"):
            shutil.copyfile(ROOT / "dev" / name, cls.dev / name)
        (cls.dev / "k8s").symlink_to(ROOT / "dev/k8s", target_is_directory=True)
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
            "if name == 'go':\n"
            "    if args == ['env', 'GOARCH']: print('amd64')\n"
            "    else: sys.exit(1)\n"
            "elif name == 'stripe':\n"
            "    if args == ['config', '--list']: print('account_id = stale')\n"
            "    elif args == ['listen', '--print-secret', '--skip-update']:\n"
            "        time.sleep(float(os.environ.get('STRIPE_TEST_DELAY', '0')))\n"
            "        print(os.environ.get('STRIPE_TEST_SECRET', ''))\n"
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
        for name in ("go", "stripe", "kubectl", "ctlptl", "minikube", "tilt", "docker", "pnpm"):
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

    def test_selected_service_includes_dependencies_without_unrelated_services(self):
        self.assertEqual(self.api_enabled, {
            "api", "api-compile", "mysql", "clickhouse", "redis",
            "namespace", "github-credentials", "uncategorized",
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
                "periodSeconds": 5,
                "failureThreshold": 30,
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
            [shutil.which("node"), str(ROOT / "dev/stripe-webhook-secret.mjs")],
            PATH=str(self.root / "missing-bin"),
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout, "")

    def test_failed_stripe_auth_or_invalid_secret_does_not_overwrite_env(self):
        for env in (
            {"FAIL_COMMAND": "stripe"},
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
            "ctrl-worker": {"mysql", "clickhouse", "restate", "registry", "rbac", "control-worker-compile"},
            "rbac": {"namespace"},
        }
        for name, dependencies in required.items():
            with self.subTest(resource=name):
                self.assertTrue(dependencies <= set(self.default[name]["ResourceDependencies"]))
        for name in ("topolvm-controller", "topolvm-node", "topolvm-lvmd-0"):
            self.assertIn("topolvm-vg", self.default[name]["ResourceDependencies"])
        for name in ("mysql", "clickhouse", "redis", "s3", "restate"):
            self.assertIn("namespace", self.default[name]["ResourceDependencies"])
        for name in ("cilium-ready", "hubble", "topolvm-vg"):
            self.assertTrue(self.default[name]["DeployTarget"]["AllowParallel"])
        self.assertIn("heimdall", self.default)
        self.assertNotIn("cilium-network-policy-crd", self.default)
        self.assertNotIn("cilium-ready", self.orb)
        self.assertIn("cilium-network-policy-crd", self.orb["krane"]["ResourceDependencies"])

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
