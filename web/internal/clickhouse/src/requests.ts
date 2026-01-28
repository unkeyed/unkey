import type { Inserter } from "./client/interface";

import { z } from "zod";

export function insertApiRequest(ch: Inserter) {
  return ch.insert({
    table: "metrics.raw_api_requests_v1",
    schema: z.object({
      request_id: z.string(),
      time: z.int(),
      workspace_id: z.string(),
      host: z.string(),
      method: z.string(),
      path: z.string(),
      request_headers: z.array(z.string()),
      request_body: z.string(),
      response_status: z.int(),
      response_headers: z.array(z.string()),
      response_body: z.string(),
      error: z.string().optional().prefault(""),
      service_latency: z.int(),
      user_agent: z.string(),
      ip_address: z.string(),
      continent: z.string().nullable().prefault(""),
      city: z.string().nullable().prefault(""),
      country: z.string().nullable().prefault(""),
      colo: z.string().nullable().prefault(""),
    }),
  });
}
