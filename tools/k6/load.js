import http from "k6/http";
import { check } from "k6";

export const options = {
  stages: [
    { duration: "10s", target: 100 },
    { duration: "2m", target: 100 },
    { duration: "10s", target: 200 },
    { duration: "2m", target: 200 },
    { duration: "10s", target: 300 },
    { duration: "2m", target: 300 },
    { duration: "10s", target: 400 },
    { duration: "2m", target: 400 },
    { duration: "10s", target: 500 },
    { duration: "2m", target: 500 },
  ],

  thresholds: {
    http_req_duration: ["p(95)<100"], // 95% of requests should be below 100ms
  },
};

export default function() {
  const res = http.post(
    "https://api.unkey.dev/v1/keys/verify",
    JSON.stringify({
      key: "XXX",
    }),
    {
      headers: {
        "Content-Type": "application/json",
      },
    },
  );
  check(res, {
    "is status 200": (r) => r.status === 200,
  });
}
