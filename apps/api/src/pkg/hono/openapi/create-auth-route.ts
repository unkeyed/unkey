import { createRoute, z } from "@hono/zod-openapi";

import { openApiErrorResponses } from "@/pkg/errors";

// extract type of RouteConfig which is not exported
type RouteConfig = Parameters<typeof createRoute>[0];

// expect headers to be zod object so we can merge with ZAuthHeader schema
type CustomRequest = Omit<NonNullable<RouteConfig["request"]>, "headers"> & {
  headers?: z.AnyZodObject;
};

type Config = Omit<RouteConfig, "request"> & {
  request?: CustomRequest;
};

export const SECURITY_SCHEME_NAME = "bearerAuth";

const ZAuthHeader = z.object({
  authorization: z.string().regex(/^Bearer [a-zA-Z0-9_]+/).openapi({
    description: "A root key to authorize the request formatted as bearer token",
    example: "Bearer unkey_1234",
  }),
});

type AuthHeaders = {
  headers: typeof ZAuthHeader;
};

export const createAuthenticatedRoute = <R extends Config>(routeConfig: R) => {
  type Request = Omit<Pick<R, "request">, "headers"> & AuthHeaders;

  const request = {
    ...(routeConfig?.request && { ...routeConfig?.request }),
    headers: routeConfig.request?.headers
      ? ZAuthHeader.merge(routeConfig.request?.headers)
      : ZAuthHeader,
  } as Request;

  return createRoute({
    ...routeConfig,
    security: [
      {
        [SECURITY_SCHEME_NAME]: [],
      },
    ],
    request,
    responses: {
      ...(routeConfig?.responses && { ...routeConfig.responses }),
      ...openApiErrorResponses,
    },
  });
};
