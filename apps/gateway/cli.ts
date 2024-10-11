import fs from "node:fs";
import * as clack from "@clack/prompts";
import { execa } from "execa";

import { faker } from "@faker-js/faker";
import { task } from "./util";
async function main() {
  clack.intro("What would you like to deploy today?");

  const defaultExportFile = await clack.text({ message: "Where is your hono app located?" });

  const subdomain = `${faker.hacker.adjective()}-${faker.hacker.adjective()}-${
    faker.science.chemicalElement().name
  }-${faker.number.int({ min: 1000, max: 9999 })}`
    .replaceAll(/\s+/g, "-")
    .toLowerCase();

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
