import { TriggerClient } from "@trigger.dev/sdk";
// import { env } from "./lib/env";

export const client = new TriggerClient({
  id: "workflows-6eiv", // from https://cloud.trigger.dev/orgs/unkey-9e78/projects/workflows-6eiv
  // apiKey: env().TRIGGER_API_KEY,
});
