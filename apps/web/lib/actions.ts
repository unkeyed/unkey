import { auth } from "@clerk/nextjs";
import { z } from "zod";

type Result<TResult> =
  | {
      result: TResult;
      error?: never;
    }
  | {
      result?: never;
      error: string;
    };

export function serverAction<TInput, TOutput = void>(opts: {
  // rome-ignore lint/suspicious/noExplicitAny: wish I knew what to type here
  input: z.ZodSchema<TInput, any, any>;
  output?: z.ZodSchema<TOutput>;
  handler: (args: { input: TInput; ctx: { tenantId: string } }) => Promise<TOutput>;
}): (formData: FormData) => Promise<Result<TOutput>> {
  const { userId, orgId } = auth();
  const tenantId = orgId ?? userId;
  if (!tenantId) {
    throw new Error("unauthorized");
  }

  return async (formData: FormData) => {
    const req: Record<string, unknown> = {};
    formData.forEach((v, k) => {
      req[k] = v;
    });

    const input = opts.input.safeParse(req);
    if (!input.success) {
      return {
        error: input.error.message,
      };
    }

    try {
      const result = await opts.handler({ input: input.data, ctx: { tenantId } });
      return { result };
    } catch (e) {
      return { error: (e as Error).message };
    }
  };
}
