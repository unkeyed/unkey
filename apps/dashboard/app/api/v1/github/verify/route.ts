import crypto from "node:crypto";
import { Buffer } from "node:buffer";
import { NextResponse } from "next/server";

import { db } from "@/lib/db";
import { env } from "@/lib/env";

import { clerkClient } from "@clerk/nextjs";
import { sha256 } from "@unkey/hash";
import { Resend } from "@unkey/resend";
import freeDomains from "free-email-domains";
import { any } from "zod";
import { Token } from "@clerk/nextjs/server";
import { TRPCError } from "@trpc/server";
import { workspacePermissions } from "@/app/(app)/settings/root-keys/[keyId]/permissions/permissions";
import { keys } from "@unkey/db/src/schema";
import { resolve } from "node:path";

const GITHUB_KEYS_URI = "https://api.github.com/meta/public_keys/secret_scanning";
const { RESEND_API_KEY } = env(); // add RESEND_API_KEY

const resend = new Resend({ apiKey: RESEND_API_KEY ?? "re_CkEcQrjA_4Y9puR6YSUyqzCf5V98FZaKd" });

type Payload = {
  token: string;
  type: string;
  url: string;
  source: string;
};
type Key = {
  key_identifier: string;
  key: string;
  is_current: boolean;
};

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
  const verify = crypto.createVerify("SHA256").update(payload.toString());
  const result = verify.verify(public_key.key, Buffer.from(signature).toString("base64"), "base64");
  return result;
};

export async function POST(request: Request) {
  // Github validate signature
  const headers = request.headers;
  const signature = headers.get("github-public-key-signature");
  const keyId = headers.get("github-public-key-identifier");
  const data = await request.json();
  if (!data) {
    throw new Error("No data found");
  }
  if (!signature) {
    throw new Error("No signature found");
  }
  if (!keyId) {
    throw new Error("No KeyID found");
  }

  // Not sure how to test this if Github is not live or whatever
  // const isGithubVerified = verify_git_signature(payload, signature ?? "", keyId ?? "");
 
    const hashedItems =await Promise.all(data.map(async (item: Payload) => {
      const token = item.token.toString();
      // console.log(token);

      const hashedToken = await sha256(token);

      // console.log(hashedToken);

      return hashedToken;
    }));
    

  const hashCheck = await hashedItems;
  console.log("HashCheck",hashCheck[0]);
  
  const isKeysFound = await db.query.keys.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.hash, hashCheck[0]), isNull(table.deletedAt)),
  });
  //check workspace for org or personal
  // if org call getOrg from clerk look this up



  const ws = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.id, isKeysFound?.forWorkspaceId), isNull(table.deletedAt)),
  });
  if (!ws) {
    throw new Error("workspace does not exist");
  }
  const users = await getUsers(ws.tenantId);

  for await (const user of users) {
    await resend.sendPaymentIssue({
      email: user.email,
      name: user.name,
      date: date,
    });
  }
  clerkClient.organizations.getOrganizationMembershipList({ organizationId: "org_2hjrDSaW55rqt6gXFhZPyjoS2DX" });
  clerkClient.users.getUser("user_2hjrDS");
  //  if user call getUser from clerk look this up
  console.log("Found Things", isKeysFound);

  // using as console log for testing
  return NextResponse.json({ github: "No Tested", payload: data, hashedData: hashCheck });

  // const users = await getUsers("org_2hjrDSaW55rqt6gXFhZPyjoS2DX");


}

// Called when a key is found to be leaked and valid
async function alertSlack(
  title: string,
  text1: string,
  text2: string,
  email: string,
): Promise<void> {
  const url = process.env.SLACK_WEBHOOK_URL_LEAKED_KEY;
  if (!url) {
    throw new Error("Missing required environment variables");
  }
  const domain = email.split("@").at(-1);
  if (!domain) {
    throw new Error("Invalid email");
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
