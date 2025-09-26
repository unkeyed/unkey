import { check } from "k6";
import http from "k6/http";
import { Trend } from "k6/metrics";

// Custom metrics
const requestLatencyTrend = new Trend("request_latency", true);

const loadZones = [
  "amazon:us:ashburn", // US East
  "amazon:us:portland", // US West
  // 'amazon:ie:dublin',      // Europe West
  "amazon:de:frankfurt", // Europe Central
  // 'amazon:sg:singapore',   // Asia Pacific
  "amazon:jp:tokyo", // Asia Pacific East
  "amazon:au:sydney", // Australia
  // 'amazon:br:sao paulo',   // South America
  "amazon:in:mumbai", // India
  // 'amazon:ca:montreal'     // Canada
];
const equalPercent = Math.floor(100 / loadZones.length);
const distribution = {};
loadZones.forEach((zone, index) => {
  distribution[zone] = {
    loadZone: zone,
    percent:
      index === loadZones.length - 1 ? 100 - equalPercent * (loadZones.length - 1) : equalPercent,
  };
});

export const options = {
  cloud: {
    project: "3788521",
    distribution: distribution,
  },
  stages: [
    { duration: "10m", target: 10 }, // 10 req/s for 1 minute
  ],
  thresholds: {
    http_req_duration: ["p(95)<500"], // 95% of requests must complete below 500ms
    checks: ["rate>0.99"], // 99% of checks must pass
  },
};

const UNKEY_ROOT_KEY = __ENV.UNKEY_ROOT_KEY;

if (!UNKEY_ROOT_KEY) {
  throw new Error("UNKEY_ROOT_KEY environment variable is required");
}

const headers = {
  "Content-Type": "application/json",
  Authorization: `Bearer ${UNKEY_ROOT_KEY}`,
};

const identifiers = ["user1", "user2", "user3", "user4", "user5"];
// biome-ignore lint/style/noDefaultExport: k6 needs a default exporet
export default function () {
  // Randomly choose between v1 and v2 (50/50 split)

  const identifier = identifiers[Math.floor(Math.random() * identifiers.length)];

  const body = JSON.stringify({
    namespace: "benchmark",
    identifier,
    limit: 1000,
    duration: 60000,
  });

  const response =
    Math.random() < 0.5
      ? http.post("https://api.unkey.dev/v1/ratelimits.limit", body, {
          headers: headers,
        })
      : http.post("https://api.unkey.com/v2/ratelimit.limit", body, {
          headers: headers,
        });

  check(response, {
    "status is 200": (r) => r.status === 200,
  });

  requestLatencyTrend.add(response.timings.duration, { url: response.request.url });
}
