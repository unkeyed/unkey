import { test } from "vitest";
import { z } from "zod";
import { zEnv } from "./env";

test("enables metrics logs if EMIT_METRICS_LOGS is not defined", (t) => {
  const env = {
    DATABASE_HOST: "",
    DATABASE_USERNAME: "",
    DATABASE_PASSWORD: "",
    DO_RATELIMIT: z.custom<DurableObjectNamespace>((ns) => typeof ns === "object"), // pretty loose check but it'll do I think
    DO_USAGELIMIT: z.custom<DurableObjectNamespace>((ns) => typeof ns === "object"),
    VAULT_URL: "http://localhost:8080",
    VAULT_TOKEN: "",
  };

  const result = zEnv.safeParse(env);
  t.expect(result.success).toBe(true);
  t.expect(result.data!.EMIT_METRICS_LOGS).toBe(true);
});
