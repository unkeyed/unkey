"use client";

import { match } from "@unkey/match";
import { AxiomDrainDetail } from "./axiom-drain-detail";
import { HttpDrainDetail } from "./http-drain-detail";
import type { Drain, DrainTelemetry } from "./types";

export function DrainDetail({ drain, ...telemetry }: { drain: Drain } & DrainTelemetry) {
  return match(drain)
    .with({ kind: "http" }, (httpDrain) => (
      <HttpDrainDetail key={httpDrain.id} drain={httpDrain} {...telemetry} />
    ))
    .with({ kind: "axiom" }, (axiomDrain) => (
      <AxiomDrainDetail key={axiomDrain.id} drain={axiomDrain} {...telemetry} />
    ))
    .exhaustive();
}
