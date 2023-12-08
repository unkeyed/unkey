import { openApiErrorResponses } from "@/pkg/errors";
import { analytics, logger, metrics, rateLimiter, usageLimiter } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

const route = createRoute({
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
              analytics: z.string().openapi({
                description: "The name of the connected analytics service",
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
  typeof route.responses[200]["content"]["application/json"]["schema"]
>;

export const registerV1Liveness = (app: App) =>
  app.openapi(route, async (c) => {
    return c.json({
      status: "we're so back",
      services: {
        metrics: metrics.constructor.name,
        logger: logger.constructor.name,
        ratelimit: rateLimiter.constructor.name,
        usagelimit: usageLimiter.constructor.name,
        analytics: analytics.client.constructor.name,
      },
    });
  });
