import type { App } from "@/pkg/hono/app";
import { createRoute, z } from "@hono/zod-openapi";

// import { rootKeyAuth } from "@/pkg/auth/root_key";
import { openApiErrorResponses } from "@/pkg/errors";
// import { buildUnkeyQuery } from "@unkey/rbac";

const route = createRoute({
  tags: ["keys"],
  operationId: "createRole",
  method: "post",
  path: "/v1/keys.createRole",
  security: [{ bearerAuth: [] }],
  request: {
    body: {
      required: true,
      content: {
        "application/json": {
          schema: z.object({
            name: z
              .string()
              .min(3)
              .regex(/^[a-zA-Z0-9_\-\.\*]+$/, {
                message:
                  "Must be at least 3 characters long and only contain alphanumeric, periods, dashes and underscores",
              })
              .openapi({
                description: "The unique name of your role. You'll use this to reference it later.",
                example: "domain.dns.manager",
              }),
            description: z.string().optional().openapi({
              description:
                "Explain what this role does. This is just for your team, your users will not see this.",
              example: "domain.dns.manager can read and write dns records for our domains.",
            }),
          }),
        },
      },
    },
  },
  responses: {
    200: {
      description: "The configuration for an api",
      content: {
        "application/json": {
          schema: z.object({
            roleId: z.string().openapi({
              description: "The id of the role. This is used internally",
              example: "role_123",
            }),
          }),
        },
      },
    },
    ...openApiErrorResponses,
  },
});

export type Route = typeof route;
export type V1KeysCreateKeyRequest = z.infer<
  (typeof route.request.body.content)["application/json"]["schema"]
>;
export type V1KeysCreateKeyResponse = z.infer<
  (typeof route.responses)[200]["content"]["application/json"]["schema"]
>;

export const registerV1KeysCreateRole = (app: App) =>
  app.openapi(route, async (c) => {
    // const req = c.req.valid("json");
    // const auth = await rootKeyAuth(
    //   c,
    //   buildUnkeyQuery(({ or }) => or("*", "api.*.create_key")),
    // );

    // TODO: emit event to tinybird
    return c.json({
      roleId: "",
    });
  });
