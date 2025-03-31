import crypto from "node:crypto";
import { db, eq, schema } from "@/lib/db";
import { vercelIntegrationEnv } from "@/lib/env";
import { z } from "zod";

export const runtime = "nodejs";

export async function POST(request: Request) {
  const env = vercelIntegrationEnv();
  if (!env) {
    return new Response("setup env", { status: 500 });
  }
  const rawBody = await request.text();
  const rawBodyBuffer = Buffer.from(rawBody, "utf-8");
  const bodySignature = sha1(rawBodyBuffer, env.VERCEL_INTEGRATION_CLIENT_SECRET);

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
        .update(schema.vercelBindings)
        .set({ deletedAtM: Date.now() })
        .where(eq(schema.vercelBindings.projectId, p.data.payload.project.id));
      break;
    }
    case "integration-configuration.removed": {
      await db
        .update(schema.vercelBindings)
        .set({ deletedAtM: Date.now() })
        .where(eq(schema.vercelBindings.integrationId, p.data.payload.configuration.id));
      await db
        .update(schema.vercelIntegrations)
        .set({ deletedAtM: Date.now() })
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
