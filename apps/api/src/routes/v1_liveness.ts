

import { createRoute, z } from "@hono/zod-openapi";
import { App } from "@/pkg/hono/app";
import { GlobalContext } from "@/pkg/context/global";

import { openApiErrorResponses } from "@/pkg/errors"
import { AxiomMetrics } from "@/pkg/metrics";

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
              example: "we're cooking"
            }),
            services: z.object({
              metrics: z.string().openapi({
                description: "The name of the connected metrics service",
                example: "AxiomMetrics"
              }),
              logger: z.string().openapi({
                description: "The name of the connected logger service",
                example: "AxiomLogger or ConsoleLogger"
              })
            })
          }),
        },
      },

    },
    ...openApiErrorResponses

  },
});


export const registerV1Liveness = (gCtx: GlobalContext, app: App) => app.openapi(route,
  async (c) => {

    return c.jsonT({
      status: "we're cooking",
      services: {
        metrics: gCtx.metrics.constructor.name,
        logger: gCtx.logger.constructor.name
      }
    })

  },
  (z) => { }

);
