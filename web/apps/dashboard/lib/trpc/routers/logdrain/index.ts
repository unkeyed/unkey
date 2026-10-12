import { t } from "../../trpc";
import { getLogdrainMetrics } from "./metrics";
import { getRecentLogdrainDeliveries } from "./recent-deliveries";

export const logdrain = t.router({
  metrics: getLogdrainMetrics,
  recentDeliveries: getRecentLogdrainDeliveries,
});
