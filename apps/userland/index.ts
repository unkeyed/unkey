import { Router } from "@unkey/framework";
import { z } from "zod";

const router = new Router();

router.register(
  router.createRoute({
    path: "/hello/:name",
    method: "get",
    request: {
      params: z.object({
        name: z.string().min(3),
      }),
    },

    responses: {
      200: {
        description: "A greeting response",
        content: {
          "application/json": {
            schema: z.object({
              greeting: z.string().openapi({
                description: "A special greeting for the user.",
                example: "Hello World",
              }),
            }),
          },
        },
      },
      429: {
        description: "Ratelimtied",
        content: {
          "application/json": {
            schema: z.object({
              error: z.string(),
            }),
          },
        },
      },
    },
    // security: [
    //  {
    //   type: ["apiKey"],
    //   in: ["header"],
    //  },
    // ],
  }),
  async (c) => {
    const params = c.req.param();
    // const { success } = await c.env.ratelimit.limit("user");
    // if (!success) {
    //  return c.json({ error: "Try again later" }, { status: 429 });
    // }

    return c.json({
      greeting: `Hello dear ${params.name}`,
    });
  },
);

// Auth protected

router
  .register(
    router.createRoute({
      path: "/protected/add",
      method: "post",
      request: {
        body: {
          content: {
            "application/json": {
              schema: z.object({
                a: z.number(),
                b: z.number(),
              }),
            },
          },
        },
      },
      responses: {
        200: {
          description: "The sum of a + b",
          content: {
            "application/json": {
              schema: z.object({
                sum: z.number(),
              }),
            },
          },
        },
        401: {
          description: "Unauthenticated",
          content: {
            "application/json": {
              schema: z.object({
                error: z.string(),
              }),
            },
          },
        },
        403: {
          description: "Unauthorized",
          content: {
            "application/json": {
              schema: z.object({
                error: z.string(),
              }),
            },
          },
        },
      },
    }),
    async (c) => {
      console.log(c.var);
      return c.json({
        sum: 2,
      });
    },
  )
  .use();

export default router;
