import crypto from "node:crypto";
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
  const clerk = await clack.group({
    _: () =>
      clack.note(
        `Head over to https://clerk.com and set up your application.
You will receive a publishable key and a secret,
which you need in to copy in the next step.`,
        "Set up Clerk",
      ),
    NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY: () =>
      clack.password({
        message: "enter your NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY",
        validate: (s) => {
          if (s.length === 0) {
            clack.log.error("Too short");
            process.exit(1);
          }
        },
      }),
    CLERK_SECRET_KEY: () =>
      clack.password({
        message: "enter your CLERK_SECRET_KEY",
      }),
  });

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
    Clerk: {
      NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY: clerk.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY,
      CLERK_SECRET_KEY: clerk.CLERK_SECRET_KEY,
      NEXT_PUBLIC_CLERK_SIGN_IN_URL: "/auth/sign-in",
      NEXT_PUBLIC_CLERK_SIGN_UP_URL: "/auth/sign-up",
    },
    Encryption: {
      ENCRYPTION_KEYS: JSON.stringify([
        { key: crypto.randomBytes(32).toString("base64"), version: 1 },
      ]),
    },
  });

  if (fs.existsSync(envPath)) {
    clack.log.warn(".env already exists, please add the variables manually");
    clack.note(env, envPath);
  } else {
    fs.writeFileSync(envPath, env);
    clack.log.step(`Wrote variables to ${envPath}`);
  }
}
