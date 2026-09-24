import { env } from "@/lib/env";
import * as Sentry from "@sentry/nextjs";
import { Resend } from "@unkey/resend";
import { getWorkOS } from "@workos-inc/authkit-nextjs";
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

  // A failed signature check is a client error and must not be retried; a
  // failure after the signature is verified is a transient downstream provider
  // error and must be retried. Answer the two with different statuses.
  let signatureVerified = false;
  try {
    const webhook = await getWorkOS().webhooks.constructEvent({
      payload,
      sigHeader,
      secret: WORKOS_WEBHOOK_SECRET,
    });
    signatureVerified = true;

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

      // Send the email before recording audience membership. Membership is the
      // "already welcomed" marker, so writing it first would permanently
      // suppress the email for a user whose send then failed. The event-scoped
      // idempotency key lets a redelivered event re-run the send without
      // sending a second email.
      await resend.sendWelcomeEmail({
        email,
        idempotencyKey: `workos-welcome:${webhook.data.id}`,
      });
      await resend.client.contacts.create({
        audienceId: RESEND_AUDIENCE_ID,
        email,
      });
    }

    return NextResponse.json({}, { status: 200 });
  } catch (err) {
    if (!signatureVerified) {
      console.error("WorkOS webhook signature verification failed:", err);
      return NextResponse.json({ error: "Invalid signature" }, { status: 400 });
    }
    // A downstream provider failure (Resend, WorkOS) is transient, so report
    // it and return 5xx to have WorkOS redeliver. The body stays generic so an
    // unauthenticated caller cannot harvest SDK error strings (URLs, audience
    // IDs, rate-limit hints) for recon.
    console.error("WorkOS webhook processing failed:", err);
    Sentry.captureException(err);
    return NextResponse.json({ error: "Webhook processing failed" }, { status: 500 });
  }
}
