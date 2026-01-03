import { ApiCheck, AssertionBuilder, Frequency } from "checkly/constructs";
import { incidentIo } from "../../../alert-channels";
import { ratelimitsV1 } from "./group";

new ApiCheck("/v1/ratelimits.limit-async", {
  name: "/v1/ratelimits.limit - async ",
  group: ratelimitsV1,
  alertChannels: [incidentIo],
  frequency: Frequency.EVERY_1M,

  degradedResponseTime: 100,
  maxResponseTime: 10000,
  request: {
    url: "https://api.unkey.dev/v1/ratelimits.limit",
    method: "POST",
    bodyType: "JSON",
    headers: [
      {
        key: "Authorization",
        value: "Bearer {{UNKEY_ROOT_KEY}}",
      },
    ],
    body: JSON.stringify({
      namespace: "checkly",
      identifier: "checkly",
      limit: 10,
      duration: 300_000,
      async: true,
    }),
    followRedirects: true,
    skipSSL: false,
    assertions: [
      AssertionBuilder.statusCode().equals(200),
      AssertionBuilder.jsonBody("$.success").equals(true),
      AssertionBuilder.jsonBody("$.limit").equals(10),
      AssertionBuilder.jsonBody("$.remaining").lessThan(10),
    ],
  },
});

new ApiCheck("/v1/ratelimits.limit-sync", {
  name: "/v1/ratelimits.limit - sync",
  alertChannels: [incidentIo],
  group: ratelimitsV1,
  frequency: Frequency.EVERY_1M,
  degradedResponseTime: 500,
  maxResponseTime: 10000,
  request: {
    url: "https://api.unkey.dev/v1/ratelimits.limit",
    method: "POST",
    bodyType: "JSON",
    headers: [
      {
        key: "Authorization",
        value: "Bearer {{UNKEY_ROOT_KEY}}",
      },
    ],
    body: JSON.stringify({
      namespace: "checkly",
      identifier: "checkly",
      limit: 10,
      duration: 300_000,
      async: false,
    }),
    followRedirects: true,
    skipSSL: false,
    assertions: [
      AssertionBuilder.statusCode().equals(200),
      AssertionBuilder.jsonBody("$.success").equals(true),
      AssertionBuilder.jsonBody("$.limit").equals(10),
      AssertionBuilder.jsonBody("$.remaining").lessThan(10),
    ],
  },
});
