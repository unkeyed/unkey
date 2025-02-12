import { Buffer } from "node:buffer";
import crypto from "node:crypto";
import type { Readable } from "node:stream";
import { auth } from "@/lib/auth/server";
import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { sha256 } from "@unkey/hash";
import { Resend } from "@unkey/resend";
import type { NextApiRequest, NextApiResponse } from "next";

export const config = {
  api: {
    bodyParser: false,
  },
  runtime: "nodejs",
};

type Key = {
  key_identifier: string;
  key: string;
  is_current: boolean;
};
type Keys = { public_keys: Key[] };
// Needs to be tested when Github is live
const verifyGitSignature = async (
  payload: string,
  signature: string,
  keyId: string,
  githubKeysUri: string,
) => {
  const gitHub = await fetch(githubKeysUri);
  if (!gitHub.ok) {
    console.error("Github verify error", gitHub.status, await gitHub.text());
    return false;
  }
  const gitBody: Keys = await gitHub.json();
  const publicKey =
    gitBody.public_keys.find((k: Key) => {
      if (k.key_identifier === keyId) {
        return k;
      }
    }) ?? null;

  if (!publicKey) {
    console.error("No public key found");
    return false;
  }
  const verify = crypto.createVerify("SHA256").update(payload);
  return !verify.verify(publicKey.key, Buffer.from(signature.toString(), "base64"));
};

export default async function handler(request: NextApiRequest, response: NextApiResponse) {
  const { RESEND_API_KEY, GITHUB_KEYS_URI } = env();

  if (!RESEND_API_KEY || !GITHUB_KEYS_URI) {
    console.error("Missing required environment variables");
    return response.status(500).json({ error: "Internal Server Error" });
  }
  const resend = new Resend({
    apiKey: RESEND_API_KEY,
  });
  const signature = request.headers["github-public-key-signature"]?.toString();
  const keyId = request.headers["github-public-key-identifier"]?.toString();
  const rawBody = await getRawBody(request);
  const data = JSON.parse(Buffer.from(rawBody).toString("utf8"));

  if (!signature || !signature || !keyId || !data) {
    return response.status(400).json({ Error: "Invalid webhook request" });
  }

  const isGithubVerified = await verifyGitSignature(
    rawBody.toString(),
    signature,
    keyId,
    GITHUB_KEYS_URI,
  );
  if (!isGithubVerified) {
    return response.status(401).json({ error: "Unauthorized" });
  }
  type FoundKeys = {
    token: string;
    source: string;
    url: string;
    type: string;
    isFound: boolean;
  };
  const foundKeys: FoundKeys[] = [];
  for (const item of data) {
    const token = item.token.toString();
    const hashedToken = await sha256(token);
    const keyFound = await db.query.keys.findFirst({
      columns: { id: true, forWorkspaceId: true },
      where: (table, { and, eq, isNull }) =>
        and(eq(table.hash, hashedToken), isNull(table.deletedAtM)),
    });
    if (!keyFound) {
      foundKeys.push({
        token: hashedToken,
        source: item.source,
        url: item.url,
        type: item.type,
        isFound: false,
      });
      return;
    }
    const ws = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(
          eq(table.id, keyFound?.forWorkspaceId ? keyFound?.forWorkspaceId : ""),
          isNull(table.deletedAtM),
        ),
    });
    if (!ws) {
      console.error("Workspace not found");
      return response.status(201).json({});
    }
    const users = await getUsers(ws.tenantId);
    const date = new Date().toDateString();
    foundKeys.push({ token, source: item.source, url: item.url, type: item.type, isFound: true });
    for await (const user of users) {
      await resend.sendLeakedKeyEmail({
        email: user.email,
        date: date,
        source: item.source,
        url: item.url,
      });
    }
    await alertSlack({
      type: item.type,
      source: item.source,
      itemUrl: item.url,
      date,
      keyId: keyFound.id,
      wsName: ws.name,
      tenantId: ws.tenantId,
      email: users[0].email,
    });
  }
  const githubResponse = foundKeys.map((key) => {
    if (!key.isFound) {
      return { token_hash: key.token, token_type: key.type, label: "false_positive" };
    }
    return { token_raw: key.token, token_type: key.type, label: "true_positive" };
  });
  return response.status(201).json([...githubResponse]);
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
  itemUrl: string;
  date: string;
  keyId: string;
  wsName: string;
  tenantId: string;
  email: string;
};

async function alertSlack({
  type,
  source,
  itemUrl,
  date,
  keyId,
  wsName,
  tenantId,
  email,
}: SlackProps): Promise<void> {
  const url = process.env.SLACK_WEBHOOK_URL_LEAKED_KEY;
  if (!url) {
    console.error("Missing required environment variables");
    return;
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
              text: `Type: ${type} \n Source: ${source} \n Date: ${date} \n URL: ${itemUrl}`,
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
  const members = await auth.getOrganizationMemberList(tenantId);
  for (const m of members.data) {
    userIds.push(m.user.id);
  }

  return await Promise.all(
    userIds.map(async (userId) => {
      const user = await auth.getUser(userId);
      const email = user!.email;

      return {
        id: user!.id,
        name: user!.firstName ?? "",
        email: email ?? "",
      };
    }),
  );
}
