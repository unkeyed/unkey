import { ApiCheck, AssertionBuilder, Frequency } from "checkly/constructs";
import { incidentIo } from "../../alert-channels";

new ApiCheck("/v1/liveness", {
  name: "Liveness",
  alertChannels: [incidentIo],
  degradedResponseTime: 10000,
  maxResponseTime: 20000,
  frequency: Frequency.EVERY_1M,
  request: {
    url: "https://api.unkey.dev/v1/liveness",
    method: "GET",
    followRedirects: true,
    skipSSL: false,
    assertions: [
      AssertionBuilder.statusCode().equals(200),
      AssertionBuilder.jsonBody("$.status").equals("we're so back"),
    ],
  },
});
