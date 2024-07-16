import crypto from "node:crypto";
import { Buffer } from "node:buffer";
import { NextResponse } from "next/server";
import { Unkey, verifyKey } from "@unkey/api";
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
const UNKEY_ROOT_KEY = process.env.UNKEY_ROOT_KEY;
const UNKEY_API_ID = process.env.UNKEY_API_ID;
if (!UNKEY_ROOT_KEY) {
  throw new Error("Missing required environment variables");
}
const GITHUB_KEYS_URI = "https://api.github.com/meta/public_keys/secret_scanning";
const { RESEND_API_KEY } = env(); // add RESEND_API_KEY
const unkey = new Unkey({ rootKey: UNKEY_ROOT_KEY ?? "", baseUrl: "https://localhost:8787" });


const resend = new Resend({ apiKey: RESEND_API_KEY ?? "re_CkEcQrjA_4Y9puR6YSUyqzCf5V98FZaKd" });


type Key = {
  key_identifier: string;
  key: string;
  is_current: boolean;
};

const verify_signature = async (payload: string, signature: string, keyID: string) => {
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
async function verifyKeys(data: any) {
  const result = data.map(async (item: any) => {
    const { result, error } = await unkey.keys.verify({
      apiId: UNKEY_API_ID,
      key: item?.token ?? "",
    });
    return await { ...item, result: result, error: error };
  });
  return await Promise.all(result);
}
export async function POST(request: Request) {
  // Working Slack Command
  //alertSlack("Leaked Key", "A key has been leaked", "Please verify the key", "test@email.com");  Working

  const body = await request.json();
  if (!body) {
    throw new Error("No body found");
  }
  // if (body.length > 0) {
  //   const keysChecked = await verifyKeys(body);

   
  // }
  const users = await getUsers("org_2hjrDSaW55rqt6gXFhZPyjoS2DX");
  return NextResponse.json({ body: body, data: users });
  // for (const item of body) {
  //   if (!item?.token) {return};
  //   const { result, error } = await unkey.keys.verify({
  //     apiId: UNKEY_API_ID,
  //     key: item?.token ?? "",
  //   });
  //   if (error) {
  //     throw new Error("Error verifying key");
  //   }
  //   if (result) {
  //     console.log(result);

  //     // await alertSlack("Leaked Key", "A key has been leaked", "Please verify the key", "adfsdfs");
  //   }
  // }

  // Github validate signature
  // const headers = request.headers;
  // const signature = headers.get("github-public-key-signature");
  // const keyId = headers.get("github-public-key-identifier");
  // const keys = await payload.map((item: Payload) => item.token.toString());
  //   if (!body) {
  //     throw new Error("No body found");
  //   }
  //   const payload = new Promise((resolve) => {
  //     resolve(body);
  //   });
  //   if (!keyId) {
  //     throw new Error("No KeyID found");
  //   }
  //   if (!signature) {
  //     throw new Error("No signature found");
  //   }
  //   // const res = true
  //   const res = true;//await verify_signature(payload.toString(), signature ?? "", keyId ?? "");
  //   if (res) {
  //     // Verify Leaked Key
  //     const validated = await Promise.all(body.map(async (item: any) => {
  //       const { result, error } = await unkey.keys.verify({apiId: UNKEY_API_ID, key: item?.token ?? ""});

  //       return {...item, result: result, error: error};
  //     }
  //     ));
  //     return NextResponse.json({body: validated });
  //   }

  // const getKey = async (keyId: string) => {
  //   const { result, error } =  await unkey.keys.get({ keyId: keyId });
  //   return Promise.resolve({ result, error });
  // };
  // const workspaceId = async () => {
  //   const {result, error} = await getKey(body[0].token);
  //   if(error){
  //     throw new TRPCError({
  //       code: "NOT_FOUND",
  //       message: "workspace not found",
  //     });
  //   }
  //   if(result?.id){

  //     return Promise.resolve(result.id);
  //   }

  // }
  //Okay so you'll want to do this.
  // getKey details which gives you the owner_id and keyId
  // Get the workspace_id from the database to see see if it's a org or not.
  // get all org users
  // send email.
  //   const workspacePromise = new Promise((resolve) => {

  //     const res = db.query.workspaces.findFirst({
  //       where: (table, { and, eq, isNull }) =>
  //         and(eq(table.id, workspaceId?.toString()), isNull(table.deletedAt)),
  //     });
  //     if (!res) {
  //       throw new TRPCError({
  //         code: "NOT_FOUND",
  //         message: "workspace not found",
  //       });
  //     }
  //     return resolve(res);
  //   });

  //  if(!workspacePromise){
  //   throw new TRPCError({
  //     code: "NOT_FOUND",
  //     message: "workspace not found",
  //   });
  //  }

  //   const user = await new Promise((resolve) => {
  //     const tempUser = getUsers("user_2hjrDSaW55rqt6gXFhZPyjoS2DX");
  //     return resolve(tempUser);
  //   });
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

const keyWorkspace = async (workspaceId: string) => {
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.id, workspaceId), isNull(table.deletedAt)),
  });

  if (!workspace) {
    throw new Error(`workspace ${workspaceId} not found`);
  }

  if (!workspace.stripeCustomerId) {
    throw new Error(`workspace ${workspaceId} has no stripe customer id`);
  }
};
