import type { IncomingHttpHeaders } from "http";
import { env } from "@/lib/env";
import type { WebhookEvent } from "@clerk/nextjs/server";
import { Resend } from "@unkey/resend";
import freeDomains from "free-email-domains";
import type { NextApiRequest, NextApiResponse } from "next";
import type { WebhookRequiredHeaders } from "svix";
import { Webhook } from "svix";
export const config = {
  runtime: "nodejs",
};

const { CLERK_WEBHOOK_SECRET, RESEND_API_KEY, RESEND_AUDIENCE_ID } = env();
export default async function handler(
  req: NextApiRequestWithSvixRequiredHeaders,
  res: NextApiResponse,
) {
  const payload = JSON.stringify(req.body);
  const headers = req.headers;
  if (!CLERK_WEBHOOK_SECRET || !RESEND_API_KEY || !RESEND_AUDIENCE_ID) {
    // just return a 400 here, it will never happen but it's good to be safe
    return res.status(400).json({ Error: "Missing environment variables" });
  }
  const wh = new Webhook(CLERK_WEBHOOK_SECRET);

  let evt: WebhookEvent;
  try {
    evt = wh.verify(payload, headers) as WebhookEvent;
  } catch (_) {
    // Don't log an error, just return a 400 because the webhook signature was invalid
    return res.status(400).json({});
  }

  const resend = new Resend({ apiKey: RESEND_API_KEY });
  const eventType = evt.type;
  if (eventType === "user.created") {
    // we only care about the first email address, so we can just grab the first one
    const email = evt.data.email_addresses[0].email_address;
    if (!email) {
      return res.status(400).json({ Error: "No email address found" });
    }
    try {
      await alertSlack(email);
      await resend.client.contacts.create({
        audience_id: RESEND_AUDIENCE_ID,
        email: email,
      });
      await resend.sendWelcomeEmail({
        email,
      });
      return res.status(200).json({});
    } catch (err) {
      return res.status(400).json({
        error: (err as Error).message,
      });
    }
  }
}

type NextApiRequestWithSvixRequiredHeaders = NextApiRequest & {
  headers: IncomingHttpHeaders & WebhookRequiredHeaders;
};

/**
 * temporary
 *
 * just a webhook to let us know about potential enterprise users by filtering out all gmail.com etc domains
 */
async function alertSlack(email: string): Promise<void> {
  const url = process.env.SLACK_WEBHOOK_URL_SIGNUP;
  if (!url) {
    return;
  }
  const domain = email.split("@").at(-1);
  if (!domain) {
    return;
  }
  if (freeDomains.includes(domain)) {
    return;
  }

  await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      text: `${email} signed up`,
      blocks: [
        {
          type: "section",
          fields: [
            {
              type: "mrkdwn",
              text: `${email} signed up`,
            },
            {
              type: "mrkdwn",
              text: `<https://${domain}>`,
            },
          ],
        },
      ],
    }),
  }).catch((err: Error) => {
    console.error(err);
  });
}
