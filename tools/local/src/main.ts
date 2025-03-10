import { execSync } from "node:child_process";
import path from "node:path";
import * as clack from "@clack/prompts";
import { bootstrapApi } from "./cmd/api";
import { bootstrapDashboard } from "./cmd/dashboard";
import { bootstrapWWW } from "./cmd/www";
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
    })) as string;
  }

  switch (app) {
    case "www": {
      await startContainers(["mysql", "planetscale"]);

      const resources = await prepareDatabase();
      !skipEnv && bootstrapWWW(resources);

      break;
    }
    case "dashboard": {
      await startContainers(["planetscale", "clickhouse", "agent", "clickhouse_migrator"]);

      const resources = await prepareDatabase();
      !skipEnv && (await bootstrapDashboard(resources));
      break;
    }

    case "api": {
      await startContainers(["planetscale", "clickhouse", "agent", "clickhouse_migrator"]);

      const resources = await prepareDatabase();
      !skipEnv && (await bootstrapApi(resources));
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

  execSync(`pnpm --dir=apps/${app} dev`, { cwd: "../..", stdio: "inherit" });

  clack.outro("Done");
  process.exit(0);
}

main().catch((err) => {
  clack.log.error(err);
});
