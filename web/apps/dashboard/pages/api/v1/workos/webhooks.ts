import { env } from "@/lib/env";
import { Resend } from "@unkey/resend";
import { WorkOS } from "@workos-inc/node";
import freeDomains from "free-email-domains";
import type { NextApiRequest, NextApiResponse } from "next";

// biome-ignore lint/style/noDefaultExport: required by next.js
export default async (req: NextApiRequest, res: NextApiResponse) => {
  if (req.method === "POST") {
    const payload = req.body;
    const sigHeader = req.headers["workos-signature"] as string | undefined;
    const { RESEND_API_KEY, RESEND_AUDIENCE_ID, WORKOS_API_KEY, WORKOS_WEBHOOK_SECRET } = env();
    if (!WORKOS_API_KEY || !WORKOS_WEBHOOK_SECRET || !RESEND_API_KEY || !RESEND_AUDIENCE_ID) {
      return res.status(400).json({ Error: "Missing environment variables" });
    }

    if (!payload || !sigHeader) {
      return res.status(400).json({ Error: "Nope" });
    }
    const workos = new WorkOS(WORKOS_API_KEY);

    const webhook = await workos.webhooks.constructEvent({
      payload: payload,
      sigHeader: sigHeader,
      secret: WORKOS_WEBHOOK_SECRET,
    });

    if (!webhook) {
      return res.status(400).json({ Error: "Invalid payload" });
    }

    if (webhook.event === "user.created") {
      const webhookData = webhook.data;

      const resend = new Resend({ apiKey: RESEND_API_KEY });

      if (!webhookData.email) {
        return res.status(400).json({ Error: "No email address found" });
      }
      try {
        await alertSlack(webhookData.email);
        await resend.client.contacts.create({
          audienceId: RESEND_AUDIENCE_ID,
          email: webhookData.email,
        });
        await resend.sendWelcomeEmail({
          email: webhookData.email,
        });
        return res.status(200).json({});
      } catch (err) {
        return res.status(400).json({
          error: (err as Error).message,
        });
      }
    }
    return res.status(200).json({});
  }
};

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
