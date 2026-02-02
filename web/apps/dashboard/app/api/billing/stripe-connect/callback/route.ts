import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { env, stripeEnv } from "@/lib/env";
import { encryptToken } from "@/lib/encryption";
import { newId } from "@unkey/id";
import { NextResponse } from "next/server";
import Stripe from "stripe";

export const runtime = "nodejs";

type OAuthState = {
  workspaceId: string;
  timestamp: number;
};

function parseState(stateParam: string | null): OAuthState | null {
  if (!stateParam) {
    return null;
  }
  try {
    const decoded = Buffer.from(stateParam, "base64").toString("utf-8");
    const parsed = JSON.parse(decoded) as unknown;
    if (
      typeof parsed === "object" &&
      parsed !== null &&
      "workspaceId" in parsed &&
      "timestamp" in parsed &&
      typeof (parsed as OAuthState).workspaceId === "string" &&
      typeof (parsed as OAuthState).timestamp === "number"
    ) {
      return parsed as OAuthState;
    }
    return null;
  } catch {
    return null;
  }
}

export async function GET(request: Request): Promise<Response> {
  const url = new URL(request.url);
  const code = url.searchParams.get("code");
  const stateParam = url.searchParams.get("state");
  const error = url.searchParams.get("error");
  const errorDescription = url.searchParams.get("error_description");

  // Parse state to get workspaceId for redirect
  const state = parseState(stateParam);
  const baseUrl = url.origin;

  // Handle OAuth errors from Stripe
  if (error) {
    console.error("Stripe Connect OAuth error:", { error, errorDescription });
    return NextResponse.redirect(
      `${baseUrl}/?error=${encodeURIComponent(errorDescription ?? error)}`,
    );
  }

  if (!code) {
    return NextResponse.redirect(
      `${baseUrl}/?error=${encodeURIComponent("Missing authorization code")}`,
    );
  }

  if (!state) {
    return NextResponse.redirect(
      `${baseUrl}/?error=${encodeURIComponent("Invalid state parameter")}`,
    );
  }

  // Validate state timestamp (5 minute expiry)
  const stateAge = Date.now() - state.timestamp;
  if (stateAge > 5 * 60 * 1000) {
    return NextResponse.redirect(
      `${baseUrl}/?error=${encodeURIComponent("Authorization request expired")}`,
    );
  }

  // Get workspace to find the slug for redirect
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq: whereEq }) => whereEq(table.id, state.workspaceId),
    columns: { id: true, slug: true },
  });

  if (!workspace) {
    return NextResponse.redirect(`${baseUrl}/?error=${encodeURIComponent("Workspace not found")}`);
  }

  const redirectUrl = `${baseUrl}/${workspace.slug}/billing/connect`;

  const stripeConfig = stripeEnv();
  const stripeClientId = process.env.STRIPE_CONNECT_CLIENT_ID;

  if (!stripeConfig || !stripeClientId) {
    console.error("Stripe Connect not configured");
    return NextResponse.redirect(
      `${redirectUrl}?error=${encodeURIComponent("Stripe Connect not configured")}`,
    );
  }

  const stripe = new Stripe(stripeConfig.STRIPE_SECRET_KEY, {
    apiVersion: "2023-10-16",
    typescript: true,
  });

  try {
    // Exchange authorization code for access token
    const response = await stripe.oauth.token({
      grant_type: "authorization_code",
      code,
    });

    if (!response.stripe_user_id || !response.access_token) {
      return NextResponse.redirect(
        `${redirectUrl}?error=${encodeURIComponent("Invalid OAuth response from Stripe")}`,
      );
    }

    // Encrypt tokens before storing using the same format as billing code
    const masterKey = process.env.MASTER_KEY;
    if (!masterKey) {
      throw new Error("MASTER_KEY environment variable not set");
    }

    const accessTokenEncrypted = encryptToken(state.workspaceId, response.access_token, masterKey);
    const refreshTokenEncrypted = encryptToken(state.workspaceId, response.refresh_token ?? "", masterKey);

    const now = Date.now();

    // Check if account already exists for this workspace
    const existingAccount = await db.query.stripeConnectedAccounts.findFirst({
      where: (table, { eq: whereEq }) => whereEq(table.workspaceId, state.workspaceId),
    });

    if (existingAccount) {
      // Update existing account (reconnecting)
      await db
        .update(schema.stripeConnectedAccounts)
        .set({
          stripeAccountId: response.stripe_user_id,
          accessTokenEncrypted: accessTokenEncrypted,
          refreshTokenEncrypted: refreshTokenEncrypted,
          scope: response.scope ?? "read_write",
          connectedAt: now,
          disconnectedAt: null,
          updatedAtM: now,
        })
        .where(eq(schema.stripeConnectedAccounts.id, existingAccount.id));

      await insertAuditLogs(db, {
        workspaceId: state.workspaceId,
        event: "stripeConnect.connect",
        actor: { type: "system", id: "stripe-oauth" },
        description: `Reconnected Stripe account ${response.stripe_user_id}`,
        resources: [
          {
            type: "stripeConnectedAccount",
            id: existingAccount.id,
            meta: { stripeAccountId: response.stripe_user_id },
          },
        ],
        context: { location: "", userAgent: request.headers.get("user-agent") ?? undefined },
      });
    } else {
      // Insert new account
      const id = newId("stripeConnectAccount");
      await db.insert(schema.stripeConnectedAccounts).values({
        id,
        workspaceId: state.workspaceId,
        stripeAccountId: response.stripe_user_id,
        accessTokenEncrypted: accessTokenEncrypted,
        refreshTokenEncrypted: refreshTokenEncrypted,
        scope: response.scope ?? "read_write",
        connectedAt: now,
        createdAtM: now,
        updatedAtM: now,
      });

      await insertAuditLogs(db, {
        workspaceId: state.workspaceId,
        event: "stripeConnect.connect",
        actor: { type: "system", id: "stripe-oauth" },
        description: `Connected Stripe account ${response.stripe_user_id}`,
        resources: [
          {
            type: "stripeConnectedAccount",
            id,
            meta: { stripeAccountId: response.stripe_user_id },
          },
        ],
        context: { location: "", userAgent: request.headers.get("user-agent") ?? undefined },
      });
    }

    return NextResponse.redirect(`${redirectUrl}?success=true`);
  } catch (err) {
    console.error("Stripe Connect OAuth token exchange failed:", err);
    const message = err instanceof Error ? err.message : "Failed to connect Stripe account";
    return NextResponse.redirect(`${redirectUrl}?error=${encodeURIComponent(message)}`);
  }
}
