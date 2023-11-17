import type { Result, VerifyKeyResult } from "@unkey/api";

declare module "h3" {
  interface H3EventContext {
    unkey?: Awaited<Result<VerifyKeyResult>>;
  }
}
