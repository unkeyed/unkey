import { env } from "@/lib/env";
import { Resend } from "@unkey/resend";
import { WorkOS } from "@workos-inc/node";
import { type NextRequest, NextResponse } from "next/server";

export async function POST(req: NextRequest) {
  const { RESEND_API_KEY, RESEND_AUDIENCE_ID, WORKOS_API_KEY, WORKOS_WEBHOOK_SECRET } = env();

  if (!WORKOS_API_KEY || !WORKOS_WEBHOOK_SECRET || !RESEND_API_KEY || !RESEND_AUDIENCE_ID) {
    return NextResponse.json({ Error: "Missing environment variables" }, { status: 400 });
  }

  const sigHeader = req.headers.get("workos-signature");
  if (!sigHeader) {
    return NextResponse.json({ Error: "Missing signature header" }, { status: 400 });
  }

  // Pass the raw body so the signature is verified against the exact bytes
  // WorkOS signed, not a re-serialized JSON object.
  const payload = await req.text();
  if (!payload) {
    return NextResponse.json({ Error: "Empty payload" }, { status: 400 });
  }

  const workos = new WorkOS(WORKOS_API_KEY);

  try {
    const webhook = await workos.webhooks.constructEvent({
      payload,
      sigHeader,
      secret: WORKOS_WEBHOOK_SECRET,
    });

    if (webhook.event === "user.created" || webhook.event === "user.updated") {
      const { email, emailVerified } = webhook.data;

      if (!email) {
        return NextResponse.json({ Error: "No email address found" }, { status: 400 });
      }

      // Sign-ups create the WorkOS user before the email code is verified, so
      // an unverified user is just an attempt, not an account. Only welcome
      // verified users: OAuth sign-ups arrive verified on user.created, and
      // magic-auth sign-ups become verified via a later user.updated event.
      if (!emailVerified) {
        return NextResponse.json({}, { status: 200 });
      }

      const resend = new Resend({ apiKey: RESEND_API_KEY });

      // user.updated fires for any profile change, so use audience membership
      // as the marker that this user was already welcomed.
      const existingContact = await resend.client.contacts.get({
        audienceId: RESEND_AUDIENCE_ID,
        email,
      });
      if (existingContact.data) {
        return NextResponse.json({}, { status: 200 });
      }
      if (existingContact.error && existingContact.error.name !== "not_found") {
        // Unknown lookup failure: bail so WorkOS retries, instead of risking
        // a duplicate welcome email.
        throw new Error(`Failed to look up Resend contact: ${existingContact.error.message}`);
      }

      await resend.client.contacts.create({
        audienceId: RESEND_AUDIENCE_ID,
        email,
      });
      await resend.sendWelcomeEmail({
        email,
      });
    }

    return NextResponse.json({}, { status: 200 });
  } catch (err) {
    // Log full error server-side. Return a generic body with a uniform status
    // so unauthenticated callers cannot distinguish signature failures from
    // downstream provider failures (Resend, WorkOS) and cannot harvest
    // SDK error strings (URLs, audience IDs, rate-limit hints) for recon.
    console.error("WorkOS webhook processing failed:", err);
    return NextResponse.json({ error: "Webhook processing failed" }, { status: 400 });
  }
}
