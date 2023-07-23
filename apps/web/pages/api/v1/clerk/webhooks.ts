import type { IncomingHttpHeaders } from "http";
import type { NextApiRequest, NextApiResponse } from "next";
import type { WebhookRequiredHeaders } from "svix";
import type { WebhookEvent } from "@clerk/nextjs/server"
import { Webhook } from "svix";


const webhookSecret: string = process.env.WEBHOOK_SECRET || "";
const loopsAPIKey: string = process.env.LOOPS_API_KEY || "";
export default async function handler(
    req: NextApiRequestWithSvixRequiredHeaders,
    res: NextApiResponse
) {
    const payload = JSON.stringify(req.body);
    const headers = req.headers;

    const wh = new Webhook(webhookSecret);

    let evt: WebhookEvent;
    try {
        evt = wh.verify(payload, headers) as WebhookEvent;

    } catch (_) {
        return res.status(400).json({});
    }
    const { id } = evt.data;

    const eventType = evt.type;
    if (eventType === "user.created") {
        // we only care about the first email address, so we can just grab the first one
        const email = evt.data.email_addresses[0].email_address;
        if (!email) {
            return res.status(400).json({});
        }
        if (!loopsAPIKey) {
            return res.status(400).json({});
        }
        try {
            const loopsResponse = await fetch("https://app.loops.so/api/v1/contacts/create", {
                method: "POST",
                headers: {
                    "headers": `Bearer ${loopsAPIKey}`,
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({ email: email }),
            });
            const json = await loopsResponse.json();
            res.status(201).json({json})

        } catch (_) {
            return res.status(400).json({});
        }

    }

}

type NextApiRequestWithSvixRequiredHeaders = NextApiRequest & {
    headers: IncomingHttpHeaders & WebhookRequiredHeaders;
};