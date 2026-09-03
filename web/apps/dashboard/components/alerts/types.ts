import type { Router } from "@/lib/trpc/routers";
import type { inferRouterInputs, inferRouterOutputs } from "@trpc/server";

type AlertOutputs = inferRouterOutputs<Router>["alerts"];
type AlertInputs = inferRouterInputs<Router>["alerts"];

export type AlertListItem = AlertOutputs["list"]["alerts"][number];
export type AlertDetailData = AlertOutputs["get"];
export type AlertSeriesData = AlertOutputs["series"];
export type AlertSeriesMetric = AlertInputs["series"]["metric"];
export type AlertMetric = AlertListItem["metric"];
export type AlertStatus = AlertListItem["status"];
