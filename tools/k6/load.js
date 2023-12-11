import { check } from "k6";
import http from "k6/http";

export const options = {
  stages: [
    { duration: "10s", target: 1 },
    { duration: "2m", target: 1 },
    { duration: "10s", target: 2 },
    { duration: "2m", target: 2 },
    { duration: "10s", target: 3 },
    { duration: "2m", target: 3 },
    { duration: "10s", target: 4 },
    { duration: "2m", target: 4 },
    { duration: "10s", target: 5 },
    { duration: "2m", target: 5 },
  ],

  thresholds: {
    http_req_duration: ["p(95)<100"], // 95% of requests should be below 100ms
  },
};

export default function () {
  const res = http.post(
    "https://api.unkey.dev/v1/keys.verifyKey",
    JSON.stringify({
      key: "rl_3ZQQJ33EAjjLU1GgJPz6H9iy",
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
