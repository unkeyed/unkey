import { openApiErrorResponses } from "@/pkg/errors";
import { logger, metrics } from "@/pkg/global";
import { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

const route = createRoute({
  method: "get",
  path: "/v1/liveness",
  responses: {
    200: {
      description: "The verification result",
      content: {
        "application/json": {
          schema: z.object({
            status: z.literal("we're cooking").openapi({
              description: "The status of the server",
              example: "we're cooking",
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
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export const registerV1Liveness = (app: App) =>
  app.openapi(route, async (c) => {
    return c.jsonT({
      status: "we're cooking",
      services: {
        metrics: metrics.constructor.name,
        logger: logger.constructor.name,
      },
    });
  });
