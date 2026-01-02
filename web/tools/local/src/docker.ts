import path from "node:path";
import { execa } from "execa";
import { task } from "./util";

export async function startContainers(services: Array<string>) {
  const cwd = path.join(__dirname, "../../../deployment");

  await task("starting docker services", async (s) => {
    for (const service of services) {
      s.message(`starting ${service}`);

      await execa("docker", ["compose", "up", "-d", service], { cwd });
    }
    s.stop("services ready");
  });
}
