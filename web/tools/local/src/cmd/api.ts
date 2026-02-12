import fs from "node:fs";
import path from "node:path";
import * as clack from "@clack/prompts";
import { marshalEnv } from "../env";

const appPath = path.join(__dirname, "../../../../apps/api");
const envPath = path.join(appPath, ".dev.vars");

export async function bootstrapApi(resources: {
  workspace: { id: string };
  api: { id: string };
}) {
  const env = marshalEnv({
    Database: {
      DATABASE_HOST: "localhost:3900",
      DATABASE_USERNAME: "unkey",
      DATABASE_PASSWORD: "password",
    },
    Bootstrap: {
      UNKEY_WORKSPACE_ID: resources.workspace.id,
      UNKEY_API_ID: resources.api.id,
    },
    Vault: {
      VAULT_URL: "http://localhost:8060",
      VAULT_TOKEN: "vault-test-token-123",
    },
    Logging: {
      EMIT_METRICS_LOGS: "false",
    },
    Clickhouse: {
      CLICKHOUSE_URL: "http://default:password@localhost:8123",
    },
  });

  if (fs.existsSync(envPath)) {
    const overrideDotEnv = await clack.select({
      message: ".env already exists, do you want to override it?",
      initialValue: false,
      options: [
        { value: false, label: "No(Add the variables manually)" },
        { value: true, label: "Yes" },
      ],
    });

    if (overrideDotEnv) {
      return writeEnvFile(env, envPath);
    }
    clack.note(env, envPath);
  } else {
    writeEnvFile(env, envPath);
  }
}

const writeEnvFile = (env: string, envPath: string) => {
  fs.writeFileSync(envPath, env);
  clack.log.step(`Wrote variables to ${envPath}`);
};
