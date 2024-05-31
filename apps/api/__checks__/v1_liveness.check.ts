import { ApiCheck, AssertionBuilder } from "checkly/constructs";

new ApiCheck("/v1/liveness", {
  name: "Liveness",
  alertChannels: [],
  degradedResponseTime: 10000,
  maxResponseTime: 20000,
  request: {
    url: "https://api.unkey.dev/v1/liveness",
    method: "GET",
    followRedirects: true,
    skipSSL: false,
    assertions: [
      AssertionBuilder.statusCode().equals(200),
      AssertionBuilder.jsonBody("$[0].id").isNotNull(),
    ],
  },
  runParallel: true,
});
