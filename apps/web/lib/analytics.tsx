import { env } from "@/lib/env";
import { Tinybird } from "@chronark/zod-bird";
import { z } from "zod";
import { NextRequest, NextResponse } from "next/server";

const tb = new Tinybird({ token: env.TINYBIRD_TOKEN });
const cookieName = "__unkey_session";

export const ingestPageView = tb.buildIngestEndpoint({
    datasource: "pageviews__v1",
    event: z.object({
        time: z.number().int(),
        userId: z.string().default(""),
        sessionId: z.string(),
        tenantId: z.string().default(""),
        path: z.string(),
    }),
});

export function getSessionId(req: NextRequest, res: NextResponse): string {

    let sessionId = req.cookies.get(cookieName)?.value;
    if (!sessionId) {
        
        sessionId = ["sess", crypto.randomUUID().replaceAll(/-/, "")].join("_");
        res.cookies.set(cookieName, sessionId);
    }

    return sessionId;
}



