import type { DrainStatus } from "../logdrain-ui";
import type { LogdrainSeries } from "./charts";

type DrainBase = {
  id: string;
  name: string;
  status: DrainStatus;
};

export type HttpDrain = DrainBase & {
  kind: "http";
  config: { url: string; format: "json" | "ndjson"; headers: string[] };
};

export type AxiomDrain = DrainBase & {
  kind: "axiom";
  config: { dataset: string };
};

export type Drain = HttpDrain | AxiomDrain;

export type RecentError = {
  time: number;
  outcome: "error" | "transient_error" | "permanent_error";
  responseStatus: number;
  responseBody: string;
  error: string;
};

export type DrainTelemetry = {
  metricsSeries?: LogdrainSeries;
  metricsLoading: boolean;
  metricsError: boolean;
  recentErrorEntries?: RecentError[];
  recentErrorsLoading: boolean;
  recentErrorsError: boolean;
};
