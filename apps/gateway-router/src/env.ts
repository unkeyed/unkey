import type { DispatchNamespace } from "@cloudflare/workers-types/experimental";
import { z } from "zod";

export const zEnv = z.object({
  VERSION: z.string().default("unknown"),
  APEX_DOMAIN: z.string().default("unkey.io"),
  DISPATCH: z.custom<DispatchNamespace>((ns) => typeof ns === "object"), // pretty loose check but it'll do I think
});

export type Env = z.infer<typeof zEnv>;
