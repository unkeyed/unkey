import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";

type AlertOutputs = inferRouterOutputs<Router>["alerts"];

export type AlertListItem = AlertOutputs["list"]["alerts"][number];
export type AlertDetailData = AlertOutputs["get"];
export type AlertTimeseriesData = AlertOutputs["timeseries"];
export type AlertMetric = AlertListItem["metric"];
export type AlertStatus = AlertListItem["status"];
