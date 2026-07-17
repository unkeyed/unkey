import { auth } from "@/lib/auth/server";
import { db } from "@/lib/db";
import { env } from "@/lib/env";
import * as Sentry from "@sentry/nextjs";
import { sha256 } from "@unkey/hash";
import { Resend } from "@unkey/resend";
import { NextResponse, after } from "next/server";
import { verifyGitSignature } from "./verify-signature";

export const runtime = "nodejs";

// A single secret reported by GitHub's secret-scanning webhook.
type ReportedToken = {
  token: string | number;
  source: string;
  url: string;
  type: string;
};

// A single token reported by GitHub, once we have classified it against our
// database. `isFound` decides the label we return to GitHub; the workspace
// fields are only present for found keys and drive the notifications.
type ClassifiedToken = {
  rawToken: string;
  hashedToken: string;
  source: string;
  url: string;
  type: string;
  isFound: boolean;
  keyId?: string;
  orgId?: string;
  wsName?: string;
};

export async function POST(request: Request) {
  const { RESEND_API_KEY, GITHUB_KEYS_URI } = env();

  if (!RESEND_API_KEY || !GITHUB_KEYS_URI) {
    console.error("Missing required environment variables");
    return NextResponse.json({ error: "Internal Server Error" }, { status: 500 });
  }
  const signature = request.headers.get("github-public-key-signature");
  const keyId = request.headers.get("github-public-key-identifier");
  const rawBody = await request.text();
  if (rawBody.length === 0) {
    return NextResponse.json({ Error: "Invalid webhook request" }, { status: 400 });
  }
  let data: ReportedToken[];
  try {
    data = JSON.parse(rawBody);
  } catch {
    return NextResponse.json({ Error: "Invalid webhook request" }, { status: 400 });
  }

  if (!signature || !keyId || !Array.isArray(data)) {
    return NextResponse.json({ Error: "Invalid webhook request" }, { status: 400 });
  }

  const isGithubVerified = await verifyGitSignature(rawBody, signature, keyId, GITHUB_KEYS_URI);
  if (!isGithubVerified) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  // Classify every reported token using only database lookups so we can answer
  // GitHub quickly. GitHub retries the webhook when the response is slow, and
  // each retry would otherwise re-send emails and Slack alerts, so the slow
  // work (WorkOS lookups, Resend, Slack) is deferred to `after` below. This
  // classification is the only work on the response path, so tokens are hashed
  // in parallel and resolved with two batched queries rather than a sequential
  // pair of lookups per token.
  const items = data.map((item) => ({
    rawToken: item.token.toString(),
    source: item.source,
    url: item.url,
    type: item.type,
  }));
  const hashedTokens = await Promise.all(items.map((item) => sha256(item.rawToken)));

  const uniqueHashes = [...new Set(hashedTokens)];
  const keyRows = uniqueHashes.length
    ? await db.query.keys.findMany({
        columns: { id: true, hash: true, forWorkspaceId: true },
        where: (table, { and, inArray, isNull }) =>
          and(inArray(table.hash, uniqueHashes), isNull(table.deletedAtM)),
      })
    : [];
  const keysByHash = new Map(keyRows.map((key) => [key.hash, key]));

  const workspaceIds = [
    ...new Set(keyRows.map((key) => key.forWorkspaceId).filter((id): id is string => Boolean(id))),
  ];
  const workspaceRows = workspaceIds.length
    ? await db.query.workspaces.findMany({
        columns: { id: true, name: true, orgId: true },
        where: (table, { and, inArray, isNull }) =>
          and(inArray(table.id, workspaceIds), isNull(table.deletedAtM)),
      })
    : [];
  const workspacesById = new Map(workspaceRows.map((ws) => [ws.id, ws]));

  const classified: ClassifiedToken[] = items.map((item, index) => {
    const hashedToken = hashedTokens[index];
    const base = {
      rawToken: item.rawToken,
      hashedToken,
      source: item.source,
      url: item.url,
      type: item.type,
    };
    const keyFound = keysByHash.get(hashedToken);
    if (!keyFound) {
      return { ...base, isFound: false };
    }
    const ws = keyFound.forWorkspaceId ? workspacesById.get(keyFound.forWorkspaceId) : undefined;
    if (!ws) {
      console.error("Workspace not found");
      return { ...base, isFound: false };
    }
    if (!ws.orgId) {
      console.error("Workspace orgId not found");
      return { ...base, isFound: false };
    }
    return { ...base, isFound: true, keyId: keyFound.id, orgId: ws.orgId, wsName: ws.name };
  });

  const githubResponse = classified.map((key) => {
    if (!key.isFound) {
      return {
        token_hash: key.hashedToken,
        token_type: key.type,
        label: "false_positive",
      };
    }
    return {
      token_raw: key.rawToken,
      token_type: key.type,
      label: "true_positive",
    };
  });

  // Notify after the response is flushed to GitHub, deduplicated within this
  // request so a single leaked key produces at most one email per user and one
  // Slack alert. Emails are additionally deduplicated across GitHub webhook
  // retries via a Resend idempotency key; Slack alerts are not, so a retried
  // delivery can still post a duplicate Slack alert.
  const found = classified.filter((key) => key.isFound);
  if (found.length > 0) {
    // Route Handlers run `after` callbacks with `onAfterTaskError: undefined`, so a
    // rejection here never reaches Sentry via instrumentation — capture it explicitly
    // or a failed leaked-key notification is lost silently.
    after(() =>
      notifyLeakedKeys(RESEND_API_KEY, found).catch((err) => {
        console.error("Failed to send leaked key notifications", err);
        Sentry.captureException(err);
      }),
    );
  }

  return NextResponse.json([...githubResponse], { status: 201 });
}

// Sends notifications for the found keys. Deduplicates by hashed token so
// GitHub reporting the same secret in multiple locations only notifies once,
// and passes a Resend idempotency key so webhook retries within the idempotency
// window do not re-send the same email.
async function notifyLeakedKeys(resendApiKey: string, found: ClassifiedToken[]): Promise<void> {
  const resend = new Resend({ apiKey: resendApiKey });
  const date = new Date().toDateString();

  const uniqueKeys = new Map<string, ClassifiedToken>();
  for (const key of found) {
    if (!uniqueKeys.has(key.hashedToken)) {
      uniqueKeys.set(key.hashedToken, key);
    }
  }

  const usersByOrg = new Map<string, { id: string; email: string; name: string }[]>();
  for (const key of uniqueKeys.values()) {
    if (!key.orgId) {
      continue;
    }
    // Isolate each key: a failure resolving one org's members (WorkOS) must not
    // abort notifications for the remaining leaked keys in the batch.
    try {
      let users = usersByOrg.get(key.orgId);
      if (!users) {
        users = await getUsers(key.orgId);
        usersByOrg.set(key.orgId, users);
      }

      const notified = new Set<string>();
      for (const user of users) {
        if (!user.email || notified.has(user.email)) {
          continue;
        }
        notified.add(user.email);
        await resend.sendLeakedKeyEmail({
          email: user.email,
          date,
          source: key.source,
          url: key.url,
          idempotencyKey: `leaked-key:${key.hashedToken}:${user.email}`,
        });
      }

      await alertSlack({
        type: key.type,
        source: key.source,
        itemUrl: key.url,
        date,
        keyId: key.keyId ?? "",
        wsName: key.wsName ?? "",
        orgId: key.orgId,
        email: users[0]?.email ?? "unknown",
      });
    } catch (err) {
      console.error(`Failed to notify for leaked key in org ${key.orgId}`, err);
      Sentry.captureException(err);
    }
  }
}

type SlackProps = {
  type: string;
  source: string;
  itemUrl: string;
  date: string;
  keyId: string;
  wsName: string;
  orgId: string;
  email: string;
};

async function alertSlack({
  type,
  source,
  itemUrl,
  date,
  keyId,
  wsName,
  orgId,
  email,
}: SlackProps): Promise<void> {
  const url = process.env.SLACK_WEBHOOK_URL_LEAKED_KEY;
  if (!url) {
    console.error("Missing required environment variables");
    return;
  }
  // Let failures propagate so the caller's per-key try/catch reports them to
  // Sentry; a swallowed error here would silently drop a leaked-key alert.
  const response = await fetch(url, {
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
              text: `Key: ${keyId} \n Workspace: ${wsName} \n Tenant: ${orgId} \n User: ${email}`,
            },
          ],
        },
      ],
    }),
  });
  if (!response.ok) {
    throw new Error(`Slack webhook responded with ${response.status}`);
  }
}

async function getUsers(orgId: string): Promise<{ id: string; email: string; name: string }[]> {
  const members = await auth.getOrganizationMemberList(orgId);

  const users = await Promise.all(
    members.data.map(async (member) => {
      try {
        const user = await auth.getUser(member.user.id);
        return user;
      } catch {
        return null;
      }
    }),
  );

  return users
    .filter((user): user is NonNullable<typeof user> => user !== null)
    .map((user) => ({
      id: user.id,
      name: user.firstName ?? "",
      email: user.email ?? "",
    }));
}
