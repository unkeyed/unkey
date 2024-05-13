import type { ZodError } from "zod";
import { generateErrorMessage } from "zod-error";
import { BaseError } from "./base";

/**
 * An object does not have the required schema.
 */
export class SchemaError extends BaseError<{ raw: unknown }> {
  public readonly retry = false;
  public readonly name = SchemaError.name;

  constructor(opts: {
    message: string;
    context?: { raw: unknown };
    cause?: BaseError;
  }) {
    super({
      ...opts,
    });
  }
  static fromZod<T>(e: ZodError<T>, raw: unknown): SchemaError {
    const message = generateErrorMessage(e.issues, {
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
    });
    return new SchemaError({
      message,
      context: {
        raw: JSON.stringify(raw),
      },
    });
  }
}
