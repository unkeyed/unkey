import { openApiErrorResponses } from "@/pkg/errors";
import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

const route = createRoute({
  deprecated: true,
  tags: ["liveness"],
  operationId: "v1.liveness",
  summary: "Health check",
  description:
    "**DEPRECATED**: This API version is deprecated. Please migrate to v2. See https://www.unkey.com/docs/api-reference/v1/migration for more information.",
  method: "get",
  path: "/v1/liveness",
  responses: {
    200: {
      description: "The configured services and their status",
      content: {
        "application/json": {
          schema: z.object({
            status: z.string().openapi({
              description: "The status of the server",
            }),
            services: z.object({
              metrics: z.string().openapi({
                description: "The name of the connected metrics service",
                example: "AxiomMetrics",
              }),
              logger: z.string().openapi({
                description: "The name of the connected logger service",
                example: "AxiomLogger or ConsoleLogger",
              }),
              ratelimit: z.string().openapi({
                description: "The name of the connected ratelimit service",
              }),
              usagelimit: z.string().openapi({
                description: "The name of the connected usagelimit service",
              }),
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type V1LivenessResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1Liveness = (app: App) =>
  app.openapi(route, async (c) => {
    const { logger, metrics, rateLimiter, usageLimiter } = c.get("services");

    return c.json({
      status: "we're so back",
      services: {
        metrics: metrics.constructor.name,
        logger: logger.constructor.name,
        ratelimit: rateLimiter.constructor.name,
        usagelimit: usageLimiter.constructor.name,
      },
    });
  });
