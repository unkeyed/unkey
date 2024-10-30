import { test } from "vitest";
import { zEnv } from "./env";

test("enables metrics logs if EMIT_METRICS_LOGS is not defined", (t) => {
  const env = {
    DATABASE_HOST: "",
    DATABASE_USERNAME: "",
    DATABASE_PASSWORD: "",
    AGENT_URL: "http://localhost:8080",
    AGENT_TOKEN: "",
    DO_RATELIMIT: {},
    DO_USAGELIMIT: {},
    KEY_MIGRATIONS: {},
  };

  const result = zEnv.safeParse(env);
  t.expect(result.success).toBe(true);
  t.expect(result.data!.EMIT_METRICS_LOGS).toBe(true);
});
