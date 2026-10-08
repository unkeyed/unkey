import { spawnSync } from "node:child_process";

const result = spawnSync("stripe", ["listen", "--print-secret", "--skip-update"], {
  encoding: "utf8",
  input: "",
  timeout: 5000,
});
const secret = result.stdout?.trim() ?? "";
if (result.status === 0 && /^whsec_[A-Za-z0-9]+$/.test(secret)) {
  process.stdout.write(secret);
}
