import path from "node:path";
import * as clack from "@clack/prompts";
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
      // TODO: andreas
      // {
      //   label: "API",
      //   value: "api",
      //   hint: "api.unkey.dev",
      // },
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
      await startContainers(["mysql", "planetscale"]);

      const resources = await prepareDatabase();
      await bootstrapDashboard(resources);
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

  clack.outro(`Done, run the following command to start developing
  
pnpm --dir=apps/${app} dev`);
  process.exit(0);
}

main().catch((err) => {
  clack.log.error(err);
});
