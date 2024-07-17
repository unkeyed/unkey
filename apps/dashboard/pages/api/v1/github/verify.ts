import { Buffer } from "node:buffer";
import crypto from "node:crypto";
import { db } from "@/lib/db";
import { env } from "@/lib/env";

import type { Readable } from "node:stream";
import { clerkClient } from "@clerk/nextjs";
import { sha256 } from "@unkey/hash";
import { Resend } from "@unkey/resend";
import type { NextApiRequest, NextApiResponse } from "next";
export const config = {
  api: {
    bodyParser: false,
  },
  runtime: "nodejs",
};
const GITHUB_KEYS_URI = process.env.GITHUB_KEYS_URI;
const { RESEND_API_KEY } = env();
if (!RESEND_API_KEY || !GITHUB_KEYS_URI) {
  throw new Error("Missing required environment variables");
}
const resend = new Resend({
  apiKey: RESEND_API_KEY,
});

type Key = {
  key_identifier: string;
  key: string;
  is_current: boolean;
};
// Needs to be tested when Github is live
const verify_git_signature = async (payload: string, signature: string, keyID: string) => {
  //Check payload
  if (!payload || payload.length === 0) {
    throw new Error("No payload found");
  }
  //Check signature
  if (!signature || signature.length === 0 || typeof signature !== "string") {
    throw new Error("No signature found");
  }
  //Check KeyID
  if (!keyID || keyID.length === 0 || typeof keyID !== "string") {
    throw new Error("No KeyID found");
  }
  //Get data from Github url
  const gitHub = await fetch(GITHUB_KEYS_URI);
  if (!gitHub.ok) {
    throw new Error("No public keys found");
  }
  const gitBody = gitHub.ok ? await gitHub.json() : undefined;
  //Get public keys from Github
  const public_key =
    gitBody.public_keys.find((k: Key) => {
      if (k.key_identifier === keyID) {
        return k;
      }
    }) ?? null;

  if (!public_key) {
    console.error("No public keys found");
  }
  // Verify signature
  const verify = crypto.createVerify("SHA256").update(payload);
  if (!verify.verify(public_key.key, Buffer.from(signature.toString(), "base64"))) {
    return false;
  }
  return true;
};

export default async function handler(request: NextApiRequest, response: NextApiResponse) {
  const signature = request.headers["github-public-key-signature"]?.toString();
  const keyId = request.headers["github-public-key-identifier"]?.toString();
  const rawBody = await getRawBody(request);
  const data = JSON.parse(Buffer.from(rawBody).toString("utf8"));

  if (!signature) {
    throw new Error("No signature found");
  }
  if (!keyId) {
    throw new Error("No KeyID found");
  }
  if (data?.length === 0) {
    throw new Error("No data found");
  }
  const isGithubVerified = await verify_git_signature(rawBody.toString(), signature, keyId);
  if (!isGithubVerified) {
    throw new Error("Github signature not verified");
  }

  for (const item of data) {
    const token = item.token.toString();
    const hashedToken = await sha256(token);
    const isKeysFound = await db.query.keys.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.hash, hashedToken), isNull(table.deletedAt)),
    });
    const ws = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(
          eq(table.id, isKeysFound?.forWorkspaceId ? isKeysFound?.forWorkspaceId : ""),
          isNull(table.deletedAt),
        ),
    });
    if (!ws) {
      throw new Error("workspace does not exist");
    }
    const users = await getUsers(ws.tenantId);
    const date = new Date().toDateString();
    for await (const user of users) {
      await resend.sendLeakedKeyEmail({
        email: user.email,
        date: date,
        source: item.source,
        type: item.type,
        url: item.url,
      });

      if (isKeysFound) {
        const props = {
          type: item.type,
          source: item.source,
          date,
          keyId: isKeysFound.id,
          wsName: ws.name,
          tenantId: ws.tenantId,
          email: user.email,
        };
        alertSlack(props);
      }
    }
  }

  return response.status(201).json({});
}
async function getRawBody(readable: Readable): Promise<Buffer> {
  const chunks = [];
  for await (const chunk of readable) {
    chunks.push(typeof chunk === "string" ? Buffer.from(chunk) : chunk);
  }
  return Buffer.concat(chunks);
}
type SlackProps = {
  type: string;
  source: string;
  date: string;
  keyId: string;
  wsName: string;
  tenantId: string;
  email: string;
};

async function alertSlack({
  type,
  source,
  date,
  keyId,
  wsName,
  tenantId,
  email,
}: SlackProps): Promise<void> {
  const url = process.env.SLACK_WEBHOOK_URL_LEAKED_KEY;
  if (!url) {
    throw new Error("Missing required environment variables");
  }
  await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      text: "Leaked Key Found",
      blocks: [
        {
          type: "section",
          fields: [
            {
              type: "mrkdwn",
              text: `Type: ${type} \n Source: ${source} \n Date: ${date} \n URL: ${url}`,
            },
            {
              type: "mrkdwn",
              text: `Key: ${keyId} \n Workspace: ${wsName} \n Tenant: ${tenantId} \n User: ${email}`,
            },
          ],
        },
      ],
    }),
  }).catch((err: Error) => {
    console.error(err);
  });
}

async function getUsers(tenantId: string): Promise<{ id: string; email: string; name: string }[]> {
  const userIds: string[] = [];
  if (tenantId.startsWith("org_")) {
    const members = await clerkClient.organizations.getOrganizationMembershipList({
      organizationId: tenantId,
    });
    for (const m of members) {
      userIds.push(m.publicUserData!.userId);
    }
  } else {
    userIds.push(tenantId);
  }

  return await Promise.all(
    userIds.map(async (userId) => {
      const user = await clerkClient.users.getUser(userId);
      const email = user.emailAddresses.at(0)?.emailAddress;
      if (!email) {
        throw new Error(`user ${user.id} does not have an email`);
      }
      return {
        id: user.id,
        name: user.firstName ?? user.username ?? "there",
        email,
      };
    }),
  );
}
