import fs from "node:fs";
import path from "node:path";
import * as clack from "@clack/prompts";
import { marshalEnv } from "../env";

const appPath = path.join(__dirname, "../../../../apps/dashboard");
const envPath = path.join(appPath, ".env");

export async function bootstrapDashboard(resources: {
  workspace: { id: string };
  api: { id: string };
  webhooksApi: { id: string };
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
      UNKEY_WEBHOOK_KEYS_API_ID: resources.webhooksApi.id,
    },
    Auth: {
      AUTH_PROVIDER: "local"
    },
    Agent: {
      AGENT_URL: "http://localhost:8080",
      AGENT_TOKEN: "agent-auth-secret",
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
