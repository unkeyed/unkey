import { check } from "k6";
import http from "k6/http";
import { Trend } from "k6/metrics";

// Custom metrics
const requestLatencyTrend = new Trend("request_latency", true);

const loadZones = [
  "amazon:us:ashburn", // US East
  //'amazon:us:portland',    // US West
  // 'amazon:ie:dublin',      // Europe West
  "amazon:de:frankfurt", // Europe Central
  // 'amazon:sg:singapore',   // Asia Pacific
  "amazon:jp:tokyo", // Asia Pacific East
  "amazon:au:sydney", // Australia
  // 'amazon:br:sao paulo',   // South America
  "amazon:in:mumbai", // India
  // 'amazon:ca:montreal'     // Canada
];

const percent = Math.floor(100 / loadZones.length);

const distribution = {};
loadZones.forEach((zone) => {
  distribution[zone] = {
    loadZone: zone,
    percent: percent,
  };
});

export const options = {
  cloud: {
    project: "3788521",
    distribution: distribution,
  },
  scenarios: {
    api_v1_keyverify: {
      executor: "constant-arrival-rate",
      rate: 10,
      timeUnit: "1s",
      duration: "5m",
      preAllocatedVUs: 10,
      maxVUs: 15,
      exec: "testV1KeyVerify",
    },
    api_v2_keyverify: {
      executor: "constant-arrival-rate",
      rate: 10,
      timeUnit: "1s",
      duration: "5m",
      startTime: "5m",
      preAllocatedVUs: 10,
      maxVUs: 15,
      exec: "testV2KeyVerify",
    },
  },
  thresholds: {
    http_req_duration: ["p(95)<500"], // 95% of requests must complete below 500ms
    checks: ["rate>0.99"], // 99% of checks must pass
  },
};

const UNKEY_ROOT_KEY = __ENV.UNKEY_ROOT_KEY;
const keys = __ENV.KEYS.split(",");

if (!UNKEY_ROOT_KEY) {
  throw new Error("UNKEY_ROOT_KEY environment variable is required");
}

if (keys.length === 0) {
  throw new Error("KEYS environment variable is required");
}

const headers = {
  "Content-Type": "application/json",
  Authorization: `Bearer ${UNKEY_ROOT_KEY}`,
};

export function testV1KeyVerify() {
  const key = keys[Math.floor(Math.random() * keys.length)];

  const response = http.post(
    "https://api.unkey.dev/v1/keys.verifyKey",
    JSON.stringify({
      key: key,
    }),
    {
      headers: headers,
      tags: { version: "v1" },
    },
  );

  check(response, {
    "status is 200": (r) => r.status === 200,
  });

  requestLatencyTrend.add(response.timings.duration, { url: response.request.url });
}

export function testV2KeyVerify() {
  const key = keys[Math.floor(Math.random() * keys.length)];

  const response = http.post(
    "https://api.unkey.com/v2/keys.verifyKey",
    JSON.stringify({
      key: key,
    }),
    {
      headers: headers,
      tags: { version: "v2" },
    },
  );

  check(response, {
    "status is 200": (r) => r.status === 200,
  });

  requestLatencyTrend.add(response.timings.duration, { url: response.request.url });
}
