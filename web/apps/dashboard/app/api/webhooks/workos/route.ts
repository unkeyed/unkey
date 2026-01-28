import { env } from "@/lib/env";
import { Resend } from "@unkey/resend";
import { WorkOS } from "@workos-inc/node";
import freeDomains from "free-email-domains";
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

  let payload: unknown;
  try {
    payload = await req.json();
  } catch (err) {
    const message = err instanceof Error ? err.message : "Invalid JSON payload";
    console.error("Failed to parse webhook payload:", message);
    return NextResponse.json({ Error: "Invalid JSON payload" }, { status: 400 });
  }
  
  const workos = new WorkOS(WORKOS_API_KEY);

  try {
    const webhook = await workos.webhooks.constructEvent({
      payload,
      sigHeader,
      secret: WORKOS_WEBHOOK_SECRET,
    });

    if (webhook.event === "user.created") {
      const webhookData = webhook.data;

      if (!webhookData.email) {
        return NextResponse.json({ Error: "No email address found" }, { status: 400 });
      }

      const resend = new Resend({ apiKey: RESEND_API_KEY });

      await alertSlack(webhookData.email);
      await resend.client.contacts.create({
        audienceId: RESEND_AUDIENCE_ID,
        email: webhookData.email,
      });
      await resend.sendWelcomeEmail({
        email: webhookData.email,
      });
    }

    return NextResponse.json({}, { status: 200 });
  } catch (err) {
    const message = err instanceof Error ? err.message : "Unknown error";
    return NextResponse.json(
      { error: message },
      { status: 400 }
    );
  }
}

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
