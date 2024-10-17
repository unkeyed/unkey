import type { Context } from "hono";
import type { ZodError, z } from "zod";

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
    return c.json(
      {
        error: {
          code: "BAD_REQUEST",
          message: parseZodErrorMessage(result.error),
        },
      },
      { status: 400 },
    );
  }
}

export function parseZodErrorMessage(err: z.ZodError): string {
  try {
    const arr = JSON.parse(err.message) as Array<{
      message: string;
      path: Array<string>;
    }>;
    const { path, message } = arr[0];
    return `${path.join(".")}: ${message}`;
  } catch {
    return err.message;
  }
}
