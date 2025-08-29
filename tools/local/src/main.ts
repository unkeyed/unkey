import { execSync } from "node:child_process";
import path from "node:path";
import * as clack from "@clack/prompts";
import { bootstrapApi } from "./cmd/api";
import { bootstrapDashboard } from "./cmd/dashboard";
import { seed } from "./cmd/seed";
import { prepareDatabase } from "./db";
import { startContainers } from "./docker";
import { run, task } from "./util";

const args = process.argv.slice(2);
const passedOptions: Record<string, string | boolean> = {};

args.forEach((arg) => {
  const [key, value] = arg.split("=");
  passedOptions[key.replace("--", "")] = value || true;
});

async function main() {
  clack.intro("Setting up Unkey locally...");

  let app = passedOptions.service;
  const skipEnv = passedOptions["skip-env"];

  if (!app) {
    app = (await clack.select({
      message: "What would you like to develop?",
      maxItems: 1,
      options: [
        {
          label: "Dashboard",
          value: "dashboard",
          hint: "app.unkey.com",
        },
        {
          label: "API",
          value: "api",
          hint: "api.unkey.dev",
        },
        {
          label: "Seed Clickhouse/DB",
          value: "seed",
          hint: "app.unkey.com",
        },
      ],
    })) as string;
  }

  switch (app) {
    case "dashboard": {
      await startContainers(["planetscale", "clickhouse", "agent"]);

      const resources = await prepareDatabase();
      !skipEnv && (await bootstrapDashboard(resources));
      break;
    }

    case "api": {
      await startContainers(["planetscale", "clickhouse", "agent"]);

      const resources = await prepareDatabase();
      !skipEnv && (await bootstrapApi(resources));
      break;
    }
    case "seed": {
      await startContainers(["planetscale", "clickhouse", "agent"]);
      // Extract workspace ID if provided
      const workspaceId = passedOptions.ws as string | undefined;

      // Call seed function with workspace ID if provided
      await seed({ ws: workspaceId });
      break;
    }

    default: {
    }
  }

  // Skip build and dev server for seed operation
  if (app !== "seed") {
    await task("Building ...", async (s) => {
      await run(`pnpm turbo run build --filter=./apps/${app}^...`, {
        cwd: path.join(__dirname, "../../../"),
      });
      s.stop("build complete");
    });
    execSync(`pnpm --dir=apps/${app} dev`, { cwd: "../..", stdio: "inherit" });
  }

  clack.outro("Done");
  process.exit(0);
}

main().catch((err) => {
  clack.log.error(err);
});
