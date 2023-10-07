import crypto from "crypto";
import { db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { NextResponse } from "next/server";
import { z } from "zod";

export async function GET(request: Request) {
  const rawBody = await request.text();
  const rawBodyBuffer = Buffer.from(rawBody, "utf-8");
  const bodySignature = sha1(rawBodyBuffer, env().VERCEL_INTEGRATION_CLIENT_SECRET);

  if (bodySignature !== request.headers.get("x-vercel-signature")) {
    return Response.json(
      {
        code: "invalid_signature",
        error: "signature didn't match",
      },
      { status: 401 },
    );
  }

  const p = payload.safeParse(JSON.parse(rawBodyBuffer.toString("utf-8")));
  if (!p.success) {
    console.error(p.error.message);
    return new Response(p.error.message, { status: 400 });
  }

  switch (p.data.type) {
    case "project.removed": {
      await db
        .delete(schema.vercelBindings)
        .where(eq(schema.vercelBindings.projectId, p.data.payload.project.id));
      break;
    }
    case "integration-configuration.removed": {
      await db
        .delete(schema.vercelBindings)
        .where(eq(schema.vercelBindings.integrationId, p.data.payload.configuration.id));
      await db
        .delete(schema.vercelIntegrations)
        .where(eq(schema.vercelIntegrations.id, p.data.payload.configuration.id));
      break;
    }

    // ...
  }

  return new Response("success", {
    status: 200,
  });
}

function sha1(data: Buffer, secret: string): string {
  return crypto.createHmac("sha1", secret).update(data).digest("hex");
}

const payload = z.discriminatedUnion("type", [
  z.object({
    type: z.literal("project.removed"),
    payload: z.object({
      team: z
        .object({
          id: z.string(),
        })
        .nullable(),
      project: z.object({
        id: z.string(),
      }),
    }),
  }),
  z.object({
    type: z.literal("integration-configuration.removed"),
    payload: z.object({
      team: z
        .object({
          id: z.string(),
        })
        .nullable(),
      configuration: z.object({
        id: z.string(),
        projects: z.array(z.string()),
      }),
    }),
  }),
]);
