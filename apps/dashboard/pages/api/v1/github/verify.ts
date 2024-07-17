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
const GITHUB_KEYS_URI = "https://api.github.com/meta/public_keys/secret_scanning";
const { RESEND_API_KEY } = env(); // add RESEND_API_KEY

const resend = new Resend({
  apiKey: RESEND_API_KEY ?? "re_CkEcQrjA_4Y9puR6YSUyqzCf5V98FZaKd",
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
  async function getRawBody(readable: Readable): Promise<Buffer> {
    const chunks = [];
    for await (const chunk of readable) {
      chunks.push(typeof chunk === "string" ? Buffer.from(chunk) : chunk);
    }
    return Buffer.concat(chunks);
  }

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
  // Worked but needs to be off while other functions are tested. Hashed with example from site so will always fail till live
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
      const text1 = `Type: ${item.type} \n Source: ${item.source} \n Date: ${date} \n URL: ${item.url}`;
      const text2 = `Key: ${isKeysFound?.id} \n Workspace: ${ws.name} \n Tenant: ${ws.tenantId} \n User: ${user.email}`;
      if (isKeysFound) {
        alertSlack("Leaked Key Found", text1, text2);
      }
    }
  }

  return response.status(201).json({});
}

async function alertSlack(title: string, text1: string, text2: string): Promise<void> {
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
      text: title,
      blocks: [
        {
          type: "section",
          fields: [
            {
              type: "mrkdwn",
              text: text1,
            },
            {
              type: "mrkdwn",
              text: text2,
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
