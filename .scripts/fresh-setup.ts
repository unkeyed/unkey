import { z } from "zod";
import { envSchema, dbSchema } from "../apps/web/lib/env";
import fs from "fs";
import path from "path";
import kleur from "kleur";
import { execa } from "execa";
const __cwd = process.cwd();

const parseEnv = (path: string) => {
  const env = {};
  const file = fs.readFileSync(path, "utf8");
  file.split("\n").forEach((line) => {
    const trimmedLine = line.trim();
    if (!trimmedLine || trimmedLine.startsWith("#")) return;

    const indexOfFirstEquals = trimmedLine.indexOf("=");
    if (indexOfFirstEquals === -1) return;

    const key = trimmedLine.substring(0, indexOfFirstEquals).trim();
    let value = trimmedLine.substring(indexOfFirstEquals + 1).trim();

    if (
      (value.startsWith("'") && value.endsWith("'")) ||
      (value.startsWith('"') && value.endsWith('"'))
    ) {
      value = value.substring(1, value.length - 1);
    }

    if (key) {
      env[key] = value;
    }
  });

  return env;
};

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
    await execa("pnpm", ["bootstrap"], {
      stdout: "pipe",
      env: {
        DATABASE_HOST: validatedWebEnv.data.DATABASE_HOST,
        DATABASE_USERNAME: validatedWebEnv.data.DATABASE_USERNAME,
        DATABASE_PASSWORD: validatedWebEnv.data.DATABASE_PASSWORD,
        TENANT_ID: validatedWebEnv.data.TENANT_ID,
      },
    }).then(({ stdout }) => {
      const workspaceIdRegex = /workspaceId:\s*(\S+)/;
      const keyAuthIdRegex = /keyAuthId:\s*(\S+)/;
      const apiIdRegex = /apiId:\s*(\S+)/;

      const workspaceIdMatch = stdout.match(workspaceIdRegex);
      const keyAuthIdMatch = stdout.match(keyAuthIdRegex);
      const apiIdMatch = stdout.match(apiIdRegex);

      const workspaceId = workspaceIdMatch ? workspaceIdMatch[1].replace(/^['"]|['"]$/g, "") : null;
      const keyAuthId = keyAuthIdMatch ? keyAuthIdMatch[1].replace(/^['"]|['"]$/g, "") : null;
      const apiId = apiIdMatch ? apiIdMatch[1].replace(/^['"]|['"]$/g, "") : null;

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

      /**
       * Overwrites agent_env with bootstrapped variables
       * If you have already bootstrapped Unkey, it will NOT update your env
       * and you will have to do it manually by going into the DB and grabbing the
       * correct vars.
       *
       * TODO: I will look into a good approach for the above ^
       */
      fs.writeFileSync(agent_env, newAgentEnv, "utf8");
    });
  } catch (error) {
    /**
     * If user already has bootstrapped Unkey, it will throw an error.
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
