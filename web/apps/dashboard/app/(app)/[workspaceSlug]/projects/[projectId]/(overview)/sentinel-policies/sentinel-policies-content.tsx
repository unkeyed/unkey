"use client";

import { collection } from "@/lib/collections";
import type { SentinelPolicy } from "@/lib/trpc/routers/deploy/environment-settings/sentinel/update-middleware";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useCallback, useMemo, useState } from "react";
import { useProjectData } from "../data-provider";
import { useOptionalProjectLayout } from "../layout-provider";
import { SentinelPolicyAddPanel } from "./components/add-panel";
import { SentinelPoliciesEmpty } from "./components/list/empty";
import { SentinelPoliciesHeader } from "./components/list/header";
import { SentinelPoliciesList } from "./components/list";

export function SentinelPoliciesContent() {
  const { environments } = useProjectData();
  const layout = useOptionalProjectLayout();

  const envA = environments.find((e) => e.slug === "production") ?? environments.at(0);
  const envB = environments.find((e) => e.id !== envA?.id) ?? environments.at(1);

  const envAId = envA?.id ?? "";
  const envBId = envB?.id ?? "";

  const [isAddPanelOpen, setIsAddPanelOpen] = useState(false);

  const { data: dataA } = useLiveQuery(
    (q) =>
      q.from({ s: collection.environmentSettings }).where(({ s }) => eq(s.environmentId, envAId)),
    [envAId],
  );

  const { data: dataB } = useLiveQuery(
    (q) =>
      q.from({ s: collection.environmentSettings }).where(({ s }) => eq(s.environmentId, envBId)),
    [envBId],
  );

  const policiesA = dataA.at(0)?.sentinelConfig?.policies ?? [];
  const policiesB = dataB.at(0)?.sentinelConfig?.policies ?? [];

  const policyCount = useMemo(() => {
    const ids = new Set([...policiesA.map((p) => p.id), ...policiesB.map((p) => p.id)]);
    return ids.size;
  }, [policiesA, policiesB]);

  const handleAddPolicy = useCallback(() => setIsAddPanelOpen(true), []);

  const handleAdd = useCallback(
    (prodPolicy: SentinelPolicy | null, previewPolicy: SentinelPolicy | null) => {
      if (prodPolicy !== null) {
        collection.environmentSettings.update(envAId, (draft) => {
          draft.sentinelConfig = {
            policies: [...(draft.sentinelConfig?.policies ?? []), prodPolicy],
          };
        });
      }
      if (previewPolicy !== null) {
        collection.environmentSettings.update(envBId, (draft) => {
          draft.sentinelConfig = {
            policies: [...(draft.sentinelConfig?.policies ?? []), previewPolicy],
          };
        });
      }
    },
    [envAId, envBId],
  );

  return (
    <div className="flex flex-col gap-5">
      <SentinelPoliciesHeader onAddPolicy={handleAddPolicy} />
      {policyCount === 0 ? (
        <SentinelPoliciesEmpty />
      ) : (
        <SentinelPoliciesList
          envAId={envAId}
          envBId={envBId}
          envASlug={envA?.slug ?? "production"}
          envBSlug={envB?.slug ?? "preview"}
          policiesA={policiesA}
          policiesB={policiesB}
          topOffset={layout?.tableDistanceToTop ?? 0}
        />
      )}
      <SentinelPolicyAddPanel
        envASlug={envA?.slug ?? "production"}
        envBSlug={envB?.slug ?? "preview"}
        isOpen={isAddPanelOpen}
        topOffset={layout?.tableDistanceToTop ?? 0}
        onClose={() => setIsAddPanelOpen(false)}
        onAdd={handleAdd}
      />
    </div>
  );
}
