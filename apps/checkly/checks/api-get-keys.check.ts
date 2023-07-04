import { ApiCheck, AssertionBuilder, Frequency } from "checkly/constructs";

import { env } from "../env";

new ApiCheck("api-get-keys", {
  name: "Get Keys from an api",
  alertChannels: [],
  frequency: Frequency.EVERY_5M,
  degradedResponseTime: 1000,
  maxResponseTime: 20000,
  request: {
    url: `${env.UNKEY_BASE_URL}/v1/apis/${env.UNKEY_API_ID}/keys`,
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
      AssertionBuilder.jsonBody("$.total").greaterThan(0),
      AssertionBuilder.jsonBody("$.keys.length").greaterThan(0),
      AssertionBuilder.jsonBody("$.keys.length").lessThan(101), // 100 is the default page size
      AssertionBuilder.jsonBody("$.keys[0].id").isNotNull(),
    ],
  },
});

new ApiCheck("api-get-keys-with-limit", {
  name: "Get Keys from an api with limit",
  alertChannels: [],
  frequency: Frequency.EVERY_5M,
  degradedResponseTime: 1000,
  maxResponseTime: 20000,
  request: {
    url: `${env.UNKEY_BASE_URL}/v1/apis/${env.UNKEY_API_ID}/keys?limit=2`,
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
      AssertionBuilder.jsonBody("$.total").greaterThan(0),
      AssertionBuilder.jsonBody("$.keys.length").greaterThan(0),
      AssertionBuilder.jsonBody("$.keys.length").lessThan(3),
      AssertionBuilder.jsonBody("$.keys[0].id").isNotNull(),
    ],
  },
});




new ApiCheck("api-get-keys-with-filter-ownerId", {
  name: "Get Keys from an api with ownerId filter",
  alertChannels: [],

  frequency: Frequency.EVERY_5M,
  degradedResponseTime: 1000,
  maxResponseTime: 20000,
  request: {
    url: `${env.UNKEY_BASE_URL}/v1/apis/${env.UNKEY_API_ID}/keys?ownerId=chronark`,
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
      AssertionBuilder.jsonBody("$.total").greaterThan(0),
      AssertionBuilder.jsonBody("$.keys.length").greaterThan(0),
      AssertionBuilder.jsonBody("$.keys.length").lessThan(101),
      AssertionBuilder.jsonBody("$.keys[*].ownerId").equals("chronark"),
    ],
  },
});
