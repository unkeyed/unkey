import { run } from "@stepci/runner";
import { expect, test } from "bun:test";

test("verify a key", async () => {
  const result = await run({
    version: "1.0",
    name: "Status Test",
    env: {
      host: "api.unkey.app",
    },
    tests: {
      example: {
        steps: [
          {
            name: "Verify a key",
            http: {
              url: "https://${{env.host}}/v1/keys/verifyKey",
              method: "POST",
              json: {
                key: "XXX",
              },
              check: {
                status: 200,
              },
            },
          },
        ],
      },
    },
  });
  expect(result.result.passed).toBeTrue();
});
