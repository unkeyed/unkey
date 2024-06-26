import { execSync } from "node:child_process";
import path from "node:path";
import * as clack from "@clack/prompts";
import { bootstrapApi } from "./cmd/api";
import { bootstrapDashboard } from "./cmd/dashboard";
import { bootstrapWWW } from "./cmd/www";
import { prepareDatabase } from "./db";
import { startContainers } from "./docker";
import { run, task } from "./util";

async function main() {
  clack.intro("Setting up Unkey locally...");

  const app = await clack.select({
    message: "What would you like to develop?",
    maxItems: 1,
    options: [
      {
        label: "Dashboard",
        value: "dashboard",
        hint: "app.unkey.com",
      },
      {
        label: "Landing page",
        value: "www",
        hint: "unkey.com",
      },
      {
        label: "API",
        value: "api",
        hint: "api.unkey.dev",
      },
    ],
  });

  switch (app) {
    case "www": {
      await startContainers(["mysql", "planetscale"]);

      const resources = await prepareDatabase();
      bootstrapWWW(resources);

      break;
    }
    case "dashboard": {
      await startContainers(["planetscale", "agent_1"]);

      const resources = await prepareDatabase();
      await bootstrapDashboard(resources);
      break;
    }

    case "api": {
      await startContainers(["planetscale", "agent_1", "agent_2", "agent_3"]);

      const resources = await prepareDatabase();
      await bootstrapApi(resources);
      break;
    }

    default: {
    }
  }

  await task("Building ...", async (s) => {
    await run(`pnpm turbo run build --filter=./apps/${app}^...`, {
      cwd: path.join(__dirname, "../../../"),
    });
    s.stop("build complete");
  });

  const runDev = await clack.confirm({
    message: "Run now?",
    active: "Yes",
    inactive: "No",
    initialValue: true,
  });
  if (runDev) {
    execSync(`pnpm --dir=apps/${app} dev`, { cwd: "../..", stdio: "inherit" });
  } else {
    clack.note(`pnpm --dir=apps/${app} dev`, `Run the ${app} later with the following command`);
  }

  clack.outro("Done");
  process.exit(0);
}

main().catch((err) => {
  clack.log.error(err);
});
