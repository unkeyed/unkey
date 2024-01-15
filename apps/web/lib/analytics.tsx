import { env } from "@/lib/env";
import { NoopTinybird, Tinybird } from "@chronark/zod-bird";
import { NextRequest } from "next/server";
import { z } from "zod";

const token = env().TINYBIRD_TOKEN;
const tb = token ? new Tinybird({ token }) : new NoopTinybird();

export const ingestPageView = tb.buildIngestEndpoint({
  datasource: "pageviews__v1",
  event: z.object({
    sessionId: z.string(),
    path: z.string(),
    time: z.number().int(),
    userId: z.string().optional(),
    tenantId: z.string().optional(),
    region: z.string().optional(),
    country: z.string().optional(),
    city: z.string().optional(),
    userAgent: z.string().optional(),
    referrer: z.string().optional(),
  }),
});

/**
 * Collects page view analytics
 * This function can not fail and will not throw. Instead errors are logged to the console.
 */
export async function collectPageViewAnalytics(args: {
  req: NextRequest;
  userId?: string;
  tenantId?: string;
}): Promise<void> {
  try {
    const host = args.req.nextUrl.host;
    if (host.startsWith("localhost") || host.startsWith("127.0.0.1")) {
      // console.debug(`not collecting analytics for ${host}`);
      return;
    }

    const ip = args.req.ip;
    if (!ip) {
      console.debug("not collecting analytics for unknown ip");
      return;
    }

    const now = new Date();
    const sessionId = toBase64(
      await crypto.subtle.digest(
        "sha-256",
        new TextEncoder().encode(
          [ip, now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()].join("-"),
        ),
      ),
    );

    // replace ids to make aggregations easier
    const path = args.req.nextUrl.pathname
      .replace(/\/(api_[a-zA-Z0-9]+)/g, "[apiId]")
      .replace(/\/(key_[a-zA-Z0-9]+)/g, "[keyId]");

    await ingestPageView({
      time: now.getTime(),
      sessionId,
      userId: args.userId,
      tenantId: args.tenantId,
      path,
      region: args.req.geo?.region,
      country: args.req.geo?.country,
      city: args.req.geo?.city,
      userAgent: args.req.headers.get("User-Agent") ?? undefined,
      referrer: args.req.headers.get("Referer") ?? undefined,
    });
  } catch (e) {
    console.error("error collecting analytics", e);
  }
}

function toBase64(buffer: ArrayBuffer): string {
  let binary = "";
  const bytes = new Uint8Array(buffer);
  const len = bytes.byteLength;
  for (let i = 0; i < len; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}
