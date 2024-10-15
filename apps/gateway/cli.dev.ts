import fs from "node:fs";
import * as clack from "@clack/prompts";
import { execa } from "execa";

import { task } from "./util";
async function main() {
  clack.intro("starting local server");

  const defaultExportFile = await clack.text({ message: "Where is your hono app located?" });

  fs.copyFileSync(defaultExportFile.toString(), "../user-worker/bundle/hono.ts");

  await task("Deploying", async (s) => {
    await execa(
      "pnpm",
      ["wrangler", "deploy", "--dispatch-namespace", "gateway_demo", "--name", subdomain],
      {
        cwd: "../user-worker",
      },
    );

    s.stop("done");
  });

  clack.outro(`Visit https://${subdomain}.unkey.app`);
}

main();
