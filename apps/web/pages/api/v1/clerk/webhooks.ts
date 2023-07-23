import type { IncomingHttpHeaders } from "http";
import type { NextApiRequest, NextApiResponse } from "next";
import type { WebhookRequiredHeaders } from "svix";
import type { WebhookEvent } from "@clerk/nextjs/server";
import { Webhook } from "svix";

const webhookSecret: string | undefined = process.env.CLERK_WEBHOOK_SECRET;
const loopsAPIKey: string | undefined = process.env.LOOPS_API_KEY;
export default async function handler(
  req: NextApiRequestWithSvixRequiredHeaders,
  res: NextApiResponse,
) {
  const payload = JSON.stringify(req.body);
  const headers = req.headers;
  if (!(webhookSecret && loopsAPIKey)) {
    // just return a 400 here, it will never happen but it's good to be safe
    return res.status(400).json({ Error: "Missing environment variables" });
  }
  const wh = new Webhook(webhookSecret);

  let evt: WebhookEvent;
  try {
    evt = wh.verify(payload, headers) as WebhookEvent;
  } catch (_) {
    // Don't log an error, just return a 400 because the webhook signature was invalid
    return res.status(400).json({});
  }

  const eventType = evt.type;
  if (eventType === "user.created") {
    // we only care about the first email address, so we can just grab the first one
    const email = evt.data.email_addresses[0].email_address;
    if (!email) {
      return res.status(400).json({ Error: "No email address found" });
    }
    try {
      const loopsResponse = await fetch("https://app.loops.so/api/v1/contacts/create", {
        method: "POST",
        headers: {
          Authorization: `Bearer ${loopsAPIKey}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ email: email, source: "clerk-signup" }),
      });
      const json = await loopsResponse.json();
      if (json.status !== "success") {
        if (json.message === "Email already on list.") {
          return res.status(201).json({});
        }
        return res.status(400).json({ Error: "Loops API Error ", jsonResponse: json });
      }
      return res.status(201).json({});
    } catch (_) {
      return res.status(400).json({});
    }
  }
}

type NextApiRequestWithSvixRequiredHeaders = NextApiRequest & {
  headers: IncomingHttpHeaders & WebhookRequiredHeaders;
};
