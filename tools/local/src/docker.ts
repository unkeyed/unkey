import { exec } from "node:child_process";
import path from "node:path";
import { task } from "./util";

export async function startContainers(services: Array<string>) {
  const cwd = path.join(__dirname, "../../../deployment");

  await task("starting docker services", async (s) => {
    for (const service of services) {
      s.message(`starting ${service}`);
      await run(`docker-compose up -d ${service}`, { cwd });
    }
    s.stop("services ready");
  });
}

async function run(cmd: string, opts: { cwd: string }) {
  await new Promise((resolve, reject) => {
    const p = exec(cmd, opts);

    p.on("exit", (code) => {
      if (code === 0) {
        resolve(code);
      } else {
        reject(code);
      }
    });
  });
}
