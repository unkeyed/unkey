import { z } from "zod";
import { envSchema, dbSchema } from "../apps/web/lib/env";
import fs from "fs";
import path from "path";
import kleur from "kleur";
import { execa } from "execa";
import { parseEnv } from "./utils";

const __cwd = process.cwd();

const runDevSetup = async () => {
  const root_node_modules = path.resolve(__cwd, "node_modules");
  const root_pkg = path.resolve(__cwd, "package.json");
  const web_env = path.resolve(__cwd, "apps/web/.env");
  const agent_env = path.resolve(__cwd, "apps/agent/.env");
  const internal_db_dir = path.resolve(__cwd, "internal/db");

  if (!fs.existsSync(root_pkg)) {
    console.error(kleur.red(`🚨 Error: package.json does not exist at ${__cwd}.`));
    process.exit(0);
  }

  if (!fs.existsSync(root_node_modules)) {
    console.error(
      kleur.yellow(`🚨 Error: Node modules do not exist, please run ${kleur.cyan("pnpm install")}`),
    );
    process.exit(0);
  }

  if (!fs.existsSync(web_env)) {
    console.error(kleur.red(`🚨 Error: .env file does not exist for ${web_env}`));
    process.exit(0);
  }

  if (!fs.existsSync(agent_env)) {
    console.error(kleur.red(`🚨 Error: .env file does not exist for ${agent_env}`));
    process.exit(0);
  }

  const validatedWebEnv = envSchema.merge(dbSchema).safeParse(parseEnv(web_env));

  if (!validatedWebEnv.success) {
    console.error(validatedWebEnv.error.message);
    process.exit(0);
  }

  const agentEnvSchema = z.object({
    DATABASE_DSN: z.string().min(1),
    UNKEY_APP_AUTH_TOKEN: z.string(),
    UNKEY_KEY_AUTH_ID: z.string(),
    UNKEY_WORKSPACE_ID: z.string(),
    UNKEY_API_ID: z.string(),
  });

  const validatedAgentEnv = agentEnvSchema.safeParse(parseEnv(agent_env));

  if (!validatedAgentEnv.success) {
    console.error(validatedAgentEnv.error.issues);
    process.exit(0);
  }

  try {
    await execa(`pnpm`, ["drizzle-kit", "push:mysql"], {
      cwd: internal_db_dir,
      stdout: "inherit",
      env: {
        DRIZZLE_DATABASE_URL: `mysql://${validatedWebEnv.data.DATABASE_USERNAME}:${validatedWebEnv.data.DATABASE_PASSWORD}@${validatedWebEnv.data.DATABASE_HOST}/${validatedWebEnv.data.DATABASE_NAME}?ssl={"rejectUnauthorized":true}`,
      },
    });
  } catch (error) {
    console.error(error);
    process.exit(0);
  }

  try {
    await execa("pnpm", ["run", "bootstrap"], {
      stdout: "pipe",
    }).then(({ stdout }) => {
      const regex = /workspaceId:\s+'(.*?)',\s+keyAuthId:\s+'(.*?)',\s+apiId:\s+'(.*?)'/;
      const match = stdout.match(regex);

      if (match) {
        const [, workspaceId, keyAuthId, apiId] = match;

        const validatedBootstrapVars = z
          .object({
            workspaceId: z.string().min(1),
            keyAuthId: z.string().min(1),
            apiId: z.string().min(1),
          })
          .safeParse({ workspaceId, keyAuthId, apiId });

        if (!validatedBootstrapVars.success) {
          console.error(validatedBootstrapVars.error.issues);
          process.exit(0);
        }

        const updatedEnvVariables: z.infer<typeof agentEnvSchema> = {
          DATABASE_DSN: validatedAgentEnv.data.DATABASE_DSN,
          UNKEY_APP_AUTH_TOKEN: validatedAgentEnv.data.UNKEY_APP_AUTH_TOKEN,
          UNKEY_API_ID: validatedBootstrapVars.data.apiId,
          UNKEY_WORKSPACE_ID: validatedBootstrapVars.data.workspaceId,
          UNKEY_KEY_AUTH_ID: validatedBootstrapVars.data.keyAuthId,
        };

        const newAgentEnv = Object.entries(updatedEnvVariables)
          .map(([key, value]) => `${key}="${value}"`)
          .join("\n");

        fs.writeFileSync(agent_env, newAgentEnv, "utf8");

        console.log(kleur.yellow(`✔ Overwrote agent env with new variables.`));
      } else {
        console.error(
          kleur.yellow(
            `🚨 Error: Something went wrong. Couldn't find regex matches for variables.`,
          ),
        );
        process.exit(0);
      }
    });
  } catch (error) {
    /**
     * If user already has bootstrapped Unkey, it will throw an error, but continue down.
     */
    console.warn(error);
  }

  try {
    await execa("pnpm", ["build"], { stdout: "inherit" });
  } catch (error) {
    // console.warn(error);
  }

  console.log(kleur.green(`✔️ Setup Completed!`));

  console.log(
    kleur.yellow(`
    Run the follow commands to spin up dev environments

    ${kleur.cyan("pnpm -F agent start")}

    ${kleur.cyan("pnpm -F web dev")}
  `),
  );
};

runDevSetup().catch((e) => {
  if (e.exitCode === 1) return;
  console.error(e);
  process.exit(1);
});
