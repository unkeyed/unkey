"use client";

import { useElapsed } from "@unkey/ui";
import { DeployIdentity } from "./deploy-identity";
import { EffectChips } from "./effect-chips";
import type { DeployView } from "./model";

export function DeployMarkerPopover({ deploy }: { deploy: DeployView }) {
  const { marker } = deploy;
  const elapsed = useElapsed(marker.readyAtMs);
  return (
    <DeployIdentity
      source={marker.source}
      deploymentId={marker.deploymentId}
      aside={<span className="shrink-0 text-gray-9">Ready {elapsed}</span>}
      className="gap-1.5 text-xs"
    >
      {deploy.effects.length > 0 && <EffectChips effects={deploy.effects} className="mt-1" />}
    </DeployIdentity>
  );
}
