import { z } from "@hono/zod-openapi";
import { Context } from "hono";
import { HTTPException } from "hono/http-exception";
import { StatusCode } from "hono/utils/http-status";
import { ZodError } from "zod";
import { generateErrorMessage } from "zod-error";
import { HonoEnv } from "../hono/env";
import { ConsoleLogger } from "../logging";
import { AxiomLogger } from "../logging/axiom";
import { QueueLogger } from "../logging/queue";

const ErrorCode = z.enum([
  "BAD_REQUEST",
  "FORBIDDEN",
  "INTERNAL_SERVER_ERROR",
  "USAGE_EXCEEDED",
  "DISABLED",
  "NOT_FOUND",
  "NOT_UNIQUE",
  "RATE_LIMITED",
  "UNAUTHORIZED",
  "PRECONDITION_FAILED",
  "INSUFFICIENT_PERMISSIONS",
]);

export function errorSchemaFactory(code: z.ZodEnum<any>) {
  return z.object({
    error: z.object({
      code: code.openapi({
        description: "A machine readable error code.",
        example: code._def.values.at(0),
      }),
      docs: z.string().openapi({
        description: "A link to our documentation with more details about this error code",
        example: `https://unkey.dev/docs/api-reference/errors/code/${code._def.values.at(0)}`,
      }),
      message: z
        .string()
        .openapi({ description: "A human readable explanation of what went wrong" }),
      requestId: z.string().openapi({
        description: "Please always include the requestId in your error report",
        example: "req_1234",
      }),
    }),
  });
}

export const ErrorSchema = z.object({
  error: z.object({
    code: ErrorCode.openapi({
      description: "A machine readable error code.",
      example: "INTERNAL_SERVER_ERROR",
    }),
    docs: z.string().openapi({
      description: "A link to our documentation with more details about this error code",
      example: "https://unkey.dev/docs/api-reference/errors/code/BAD_REQUEST",
    }),
    message: z.string().openapi({ description: "A human readable explanation of what went wrong" }),
    requestId: z.string().openapi({
      description: "Please always include the requestId in your error report",
      example: "req_1234",
    }),
  }),
});

export type ErrorResponse = z.infer<typeof ErrorSchema>;

function codeToStatus(code: z.infer<typeof ErrorCode>): StatusCode {
  switch (code) {
    case "BAD_REQUEST":
      return 400;
    case "FORBIDDEN":
    case "DISABLED":
    case "UNAUTHORIZED":
    case "INSUFFICIENT_PERMISSIONS":
    case "USAGE_EXCEEDED":
      return 403;
    case "NOT_FOUND":
      return 404;
    case "NOT_UNIQUE":
      return 409;
    case "PRECONDITION_FAILED":
      return 412;
    case "RATE_LIMITED":
      return 429;
    case "INTERNAL_SERVER_ERROR":
      return 500;
  }
}

export class UnkeyApiError extends HTTPException {
  public readonly code: z.infer<typeof ErrorCode>;

  constructor({ code, message }: { code: z.infer<typeof ErrorCode>; message: string }) {
    super(codeToStatus(code), { message });
    this.code = code;
  }
}

export function handleZodError(
  result:
    | {
        success: true;
        data: any;
      }
    | {
        success: false;
        error: ZodError;
      },
  c: Context,
) {
  if (!result.success) {
    return c.json<z.infer<typeof ErrorSchema>>(
      {
        error: {
          code: "BAD_REQUEST",
          docs: "https://unkey.dev/docs/api-reference/errors/code/BAD_REQUEST",
          message: generateErrorMessage(result.error.issues, {
            maxErrors: 1,
            delimiter: {
              component: ": ",
            },
            path: {
              enabled: true,
              type: "objectNotation",
              label: "",
            },
            code: {
              enabled: true,
              label: "",
            },
            message: {
              enabled: true,
              label: "",
            },
            suffix: ', See "https://unkey.dev/docs/api-reference" for more details',
          }),
          requestId: c.get("requestId"),
        },
      },
      { status: 400 },
    );
  }
}

export function handleError(err: Error, c: Context<HonoEnv>): Response {
  const logger = c.env.LOGS
    ? new QueueLogger({ queue: c.env.LOGS })
    : c.env.AXIOM_TOKEN
      ? new AxiomLogger({ axiomToken: c.env.AXIOM_TOKEN, environment: c.env.ENVIRONMENT })
      : new ConsoleLogger();
  if (err instanceof UnkeyApiError) {
    if (err.status >= 500) {
      logger.error(err.message, {
        name: err.name,
        code: err.code,
        status: err.status,
      });
    }
    return c.json<z.infer<typeof ErrorSchema>>(
      {
        error: {
          code: err.code,
          docs: `https://unkey.dev/docs/api-reference/errors/code/${err.code}`,
          message: err.message,
          requestId: c.get("requestId"),
        },
      },
      { status: err.status },
    );
  }

  logger.error("unhandled exception in hono", {
    name: err.name,
    message: err.message,
    requestId: c.get("requestId"),
  });
  return c.json<z.infer<typeof ErrorSchema>>(
    {
      error: {
        code: "INTERNAL_SERVER_ERROR",
        docs: "https://unkey.dev/docs/api-reference/errors/code/INTERNAL_SERVER_ERROR",
        message: "something unexpected happened",
        requestId: c.get("requestId"),
      },
    },
    { status: 500 },
  );
}

export function errorResponse(c: Context, code: z.infer<typeof ErrorCode>, message: string) {
  return c.json<z.infer<typeof ErrorSchema>>(
    {
      error: {
        code: code,
        docs: `https://unkey.dev/docs/api-reference/errors/code/${code}`,
        message,
        requestId: c.get("requestId"),
      },
    },
    { status: codeToStatus(code) },
  );
}
