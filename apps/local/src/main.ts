import { exec } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import * as clack from "@clack/prompts";
import { type Api, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { drizzle } from "drizzle-orm/mysql2";
import mysql from "mysql2/promise";

async function main() {
  clack.intro("Setting up Unkey locally...");

  await task("Starting containers", async (s) => {
    await new Promise((resolve, reject) => {
      const p = exec("docker compose up -d || docker-compose up -d", {
        cwd: path.dirname(__dirname),
      });

      p.on("exit", (code) => {
        if (code === 0) {
          resolve(code);
        } else {
          reject(code);
        }
      });
    });

    s.message("Waiting a bit longer to make sure");
    await new Promise((r) => setTimeout(r, 10_000));
    s.stop("Containers ready");
  });

  const db = await task("Connecting to database", async (s) => {
    const conn = await mysql
      .createConnection("mysql://unkey:password@localhost:3306/unkey")
      .catch((err) => {
        clack.log.error(err.message);
        throw err;
      });

    s.message("pinging database");
    await conn.ping();
    s.stop("connected to database");
    return drizzle(conn, { schema, mode: "default" });
  });

  await task("migrating tabels", async (s) => {
    const cwd = path.join(__dirname, "../../../internal/db");

    await new Promise((resolve, reject) => {
      const p = exec("pnpm drizzle-kit push:mysql", {
        env: {
          DRIZZLE_DATABASE_URL: "mysql://unkey:password@localhost:3306/unkey",
          ...process.env,
        },

        cwd,
      });
      p.on("exit", (code) => {
        if (code === 0) {
          resolve(code);
        } else {
          reject(code);
        }
      });
    });
    s.stop("table migration complete");
  });

  // fs.unlinkSync(migrationsFolder);

  const resources = await task("Seeding database", async (s) => {
    const existingWorkspaces = await db.query.workspaces.findMany();
    if (existingWorkspaces.length > 0) {
      s.stop("skipping, db is already seeded");
      return null;
    }

    const workspace = {
      id: newId("workspace"),
      tenantId: "user_REPLACE_ME_LATER",
      name: "Unkey",
      internal: true,
      betaFeatures: {},
      features: {},
      createdAt: new Date(),
    };
    await db.insert(schema.workspaces).values(workspace);
    s.message("Created root workspace");

    const keyAuth = {
      id: newId("keyAuth"),
      workspaceId: workspace.id,
      createdAt: new Date(),
    };

    await db.insert(schema.keyAuth).values(keyAuth);
    s.message("Created root keyspace");

    /**
     * Set up an api for production
     */
    const api = {
      id: newId("api"),
      name: "preview",
      workspaceId: workspace.id,
      authType: "key",
      keyAuthId: keyAuth.id,
      createdAt: new Date(),
      deletedAt: null,
      ipWhitelist: null,
    } satisfies Api;
    await db.insert(schema.apis).values(api);
    s.message("Created root api");

    s.stop("seed done");
    return { workspace, api };
  });

  //   console.info("Add these to /apps/api/.dev.vars and /apps/dashboard/.env");
  //   console.info(`
  // UNKEY_WORKSPACE_ID="${workspaceId}"
  // UNKEY_API_ID="${apiId}"
  // UNKEY_KEY_AUTH_ID="${keyAuth.id}"
  //     `);

  await clack.note(
    `Setup your clerk app now.
  header over to clekr.com TODO:
  
  `,
    "Clerk setup",
  );

  const clerkData = await clack.group({
    NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY: () =>
      clack.password({ message: "NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY" }),
    CLERK_SECRET_KEY: () => clack.password({ message: "CLERK_SECRET_KEY" }),
  });

  if (resources) {
    const envContent = `# Database    
DATABASE_HOST="localhost:3900"
DATABASE_USERNAME="unkey"
DATABASE_PASSWORD="password"

# Bootstrap
UNKEY_WORKSPACE_ID="${resources.workspace.id}"
UNKEY_API_ID="${resources.api.id}"
    `;

    clack.log.info(`Your environment variables:

${envContent}`);

    const writeDotEnv = await clack.confirm({
      message: "Do you want to create the required .env files automatically?",
      active: "yes",
      inactive: "no",
      initialValue: true,
    });

    if (writeDotEnv) {
      const dashboardDotEnvPath = path.join(__dirname, "../../dashboard/.env");
      const dashboardDotEnvExists = fs.existsSync(dashboardDotEnvPath);
      if (dashboardDotEnvExists) {
        clack.log.warn(`${dashboardDotEnvPath} already exists, please add the variables manually`);
      } else {
        fs.writeFileSync(dashboardDotEnvPath, envContent);
        fs.appendFileSync(
          dashboardDotEnvPath,
          `
NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY="${clerkData.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY}"
CLERK_SECRET_KEY="${clerkData.CLERK_SECRET_KEY}"
      `,
        );
        clack.log.step(`Wrote variables to ${dashboardDotEnvPath}`);
      }

      const apiDotEnvPath = path.join(__dirname, "../../api/.dev.vars");
      const apiDotEnvExists = fs.existsSync(apiDotEnvPath);
      if (apiDotEnvExists) {
        clack.log.warn(`${apiDotEnvPath} already exists, please add the variables manually`);
      } else {
        fs.writeFileSync(apiDotEnvPath, envContent);
        clack.log.step(`Wrote variables to ${apiDotEnvPath}`);
      }

      const wwwDotEnvPath = path.join(__dirname, "../../www/.env");
      const wwwDotEnvExists = fs.existsSync(wwwDotEnvPath);
      if (wwwDotEnvExists) {
        clack.log.warn(`${wwwDotEnvPath} already exists, please add the variables manually`);
      } else {
        fs.writeFileSync(wwwDotEnvPath, envContent);
        clack.log.step(`Wrote variables to ${wwwDotEnvPath}`);
      }
    }
  }

  const app = await clack.select({
    message: "What would you like to develop?",
    maxItems: 1,
    options: [
      {
        label: "Dashboard",
        value: "./apps/dashboard",
        hint: "app.unkey.com",
      },
      {
        label: "Landing page",
        value: "./apps/www",
        hint: "unkey.com",
      },
      {
        label: "API",
        value: "./apps/api",
        hint: "api.unkey.dev",
      },
    ],
  });
  clack.outro(`Done, run the following command to start developing
  
pnpm --dir=${app} dev`);
  process.exit(0);
}

main().catch((err) => {
  clack.log.error(err);
});

async function task<T>(
  name: string,
  fn: (s: ReturnType<typeof clack.spinner>) => Promise<T>,
): Promise<T> {
  const spinner = clack.spinner();
  spinner.start(name);

  try {
    const res = await fn(spinner);
    return res;
  } catch (err) {
    spinner.stop((err as Error).message);
    process.exit(1);
    // just to make ts happy
    return undefined as T;
  }
}
