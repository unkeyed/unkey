import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

import { rootKeyAuth } from "@/pkg/auth/root_key";
import { openApiErrorResponses } from "@/pkg/errors";

const route = createRoute({
  method: "post",
  path: "/v1/billing.ingestEvent",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            idempotencyId: z.string().optional().openapi({
              description: "Idempotency id",
            }),
            namespace: z.string().optional().default("default").openapi({
              description:
                "Namespaces group different limits together for better analytics. You might have a namespace for your public API and one for internal tRPC routes.",
              example: "email.outbound",
            }),
            tenantId: z.string().openapi({
              description: "Identifier of your user or workspace.",
              example: "user_123",
            }),
            cost: z.number().int().min(1).default(1).optional().openapi({
              description:
                "Expensive requests may use up more tokens. You can specify a cost to the request here and we'll deduct this many tokens in the current window. If there are not enough tokens left, the request is denied.",
              example: 2,
              default: 1,
            }),
            meta: z
              .record(z.union([z.string(), z.boolean(), z.number(), z.null()]))
              .optional()
              .openapi({
                description: "Attach any metadata to this request",
              }),
            resources: z
              .array(
                z.object({
                  type: z.string().openapi({
                    description: "The type of resource",
                    example: "organization",
                  }),
                  id: z.string().openapi({
                    description: "The unique identifier for the resource",
                    example: "org_123",
                  }),
                  name: z.string().optional().openapi({
                    description: "A human readable name for this resource",
                    example: "unkey",
                  }),
                  meta: z
                    .record(z.union([z.string(), z.boolean(), z.number(), z.null()]))
                    .optional()
                    .openapi({
                      description: "Attach any metadata to this resources",
                    }),
                }),
              )
              .optional()
              .openapi({
                description: "Resources that are about to be accessed by the user",
                example: [
                  {
                    type: "project",
                    id: "p_123",
                    name: "dub",
                  },
                ],
              }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "",
      content: {
        "application/json": {
          schema: z.object({
            ack: z.boolean().openapi({
              description: "Simply acknowledges that this event was durably stored",
              example: true,
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});
export type Route = typeof route;

export type V1BillingIngestEventRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1BillingIngestEventResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1BillingIngestEvent = (app: App) =>
  app.openapi(route, async (c) => {
    const req = c.req.valid("json");
    const rootKey = await rootKeyAuth(c);

    console.log(
      await c.env.BILLING.writeDataPoint({
        blobs: [req.tenantId, req.idempotencyId ?? crypto.randomUUID()],
        doubles: [req.cost ?? 1],
        indexes: [rootKey.authorizedWorkspaceId],
      }),
    );

    return c.json({
      ack: true,
    });
  });
