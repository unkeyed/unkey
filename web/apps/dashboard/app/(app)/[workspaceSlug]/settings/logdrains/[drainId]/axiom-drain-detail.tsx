"use client";

import { Button, Input, SettingCard } from "@unkey/ui";
import { useEffect, useState } from "react";
import { DrainShell } from "./drain-shell";
import type { AxiomDrain, DrainTelemetry } from "./types";
import { useDrainUpdate } from "./use-drain-update";

export function AxiomDrainDetail({ drain, ...telemetry }: { drain: AxiomDrain } & DrainTelemetry) {
  const [destination, setDestination] = useState(drain.config.dataset);
  const [token, setToken] = useState("");
  const update = useDrainUpdate(drain.id, (variables) => {
    if (variables.destination?.kind === "axiom" && variables.destination.config.token) {
      setToken("");
    }
  });

  useEffect(() => setDestination(drain.config.dataset), [drain.config.dataset]);

  return (
    <DrainShell
      drain={drain}
      destination={destination}
      currentDestination={drain.config.dataset}
      onDestinationChange={setDestination}
      onSaveDestination={(dataset) =>
        update.mutate({ id: drain.id, destination: { kind: "axiom", config: { dataset } } })
      }
      update={update}
      {...telemetry}
    >
      <SettingCard
        title="Token"
        description="Enter a new Axiom API token to replace the current token."
        contentWidth="w-full lg:w-[420px] justify-end"
      >
        <Input
          type="password"
          aria-label="Axiom API token"
          value={token}
          placeholder="Enter a new token"
          onChange={(event) => setToken(event.target.value)}
        />
        <Button
          variant="primary"
          loading={update.isLoading}
          disabled={!token.trim()}
          onClick={() =>
            update.mutate({
              id: drain.id,
              destination: { kind: "axiom", config: { token: token.trim() } },
            })
          }
        >
          Save token
        </Button>
      </SettingCard>
    </DrainShell>
  );
}
