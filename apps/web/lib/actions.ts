import { auth } from "@clerk/nextjs";
import { z } from "zod";

import { Result, result } from "@unkey/result";

export function serverAction<TInput, TOutput = void>(opts: {
  input: z.ZodSchema<TInput, any, any>;
  output?: z.ZodSchema<TOutput>;
  handler: (args: {
    input: TInput;
    ctx: { tenantId: string; userId: string };
  }) => Promise<TOutput>;
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
      return result.fail(input.error);
    }

    try {
      const res = await opts.handler({
        input: input.data,
        ctx: { tenantId, userId: userId! },
      });
      return result.success(res);
    } catch (e) {
      return result.fail({ message: (e as Error).message });
    }
  };
}
