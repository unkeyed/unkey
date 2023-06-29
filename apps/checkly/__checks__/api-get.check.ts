import { ApiCheck, AssertionBuilder, Frequency } from "checkly/constructs";

import { env } from "./env";

new ApiCheck("api-get", {
  name: "Get information about an API",
  alertChannels: [],
  frequency: Frequency.EVERY_1M,
  degradedResponseTime: 1000,
  maxResponseTime: 20000,
  request: {
    url: `${env.UNKEY_BASE_URL}/v1/apis/${env.UNKEY_API_ID}`,
    method: "GET",
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
      AssertionBuilder.jsonBody("$.id").equals(env.UNKEY_API_ID),
      AssertionBuilder.jsonBody("$.name").equals("checkly-e2e"),
      AssertionBuilder.jsonBody("$.workspaceId").isNotNull(),
    ],
  },
});
