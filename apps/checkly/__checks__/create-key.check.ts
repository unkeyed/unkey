import { ApiCheck, AssertionBuilder, Frequency } from "checkly/constructs";

import { env } from "./env";

new ApiCheck("create-key", {
  name: "Create a new key with defaults",
  alertChannels: [],
  frequency: Frequency.EVERY_1M,
  degradedResponseTime: 1000,
  maxResponseTime: 20000,
  request: {
    url: `${env.UNKEY_BASE_URL}/v1/keys`,
    method: "POST",
    body: JSON.stringify({
      apiId: env.UNKEY_API_ID,
    }),
    headers: [
      {
        key: "Authorization",
        value: `Bearer ${env.UNKEY}`,
      },
    ],
    followRedirects: true,
    skipSSL: false,
    assertions: [
      AssertionBuilder.statusCode().equals(200),
      AssertionBuilder.jsonBody("$.key").isNotNull(),
    ],
  },
});
