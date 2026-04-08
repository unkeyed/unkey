"use client";

import { collection } from "@/lib/collections";
import type { SentinelPolicy } from "@/lib/collections/deploy/sentinel-policies.schema";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useCallback, useState } from "react";
import { useProjectData } from "../data-provider";
import { useOptionalProjectLayout } from "../layout-provider";
import { SentinelPolicyAddPanel } from "./components/add-panel";
import { SentinelPoliciesList } from "./components/list";
import { SentinelPoliciesEmpty } from "./components/list/empty";
import { SentinelPoliciesHeader } from "./components/list/header";
import { SentinelPoliciesUnsavedBar } from "./components/list/sentinel-policies-unsaved-bar";
import { useSentinelDraft } from "./components/list/use-sentinel-draft";

export function SentinelPoliciesContent() {
  const { environments } = useProjectData();
  const layout = useOptionalProjectLayout();

  const envA = environments.find((e) => e.slug === "production") ?? environments.at(0);
  const envB = environments.find((e) => e.id !== envA?.id) ?? environments.at(1);

  const envAId = envA?.id ?? "";
  const envBId = envB?.id ?? "";

  const [isAddPanelOpen, setIsAddPanelOpen] = useState(false);

  const { data: rowsA } = useLiveQuery(
    (q) => q.from({ p: collection.sentinelPolicies }).where(({ p }) => eq(p.environmentId, envAId)),
    [envAId],
  );

  const { data: rowsB } = useLiveQuery(
    (q) => q.from({ p: collection.sentinelPolicies }).where(({ p }) => eq(p.environmentId, envBId)),
    [envBId],
  );

  // Strip the row-only `environmentId` field — downstream draft logic operates
  // on bare SentinelPolicy values.
  const policiesA: SentinelPolicy[] = rowsA.map(
    ({ environmentId: _e, ...p }) => p as SentinelPolicy,
  );
  const policiesB: SentinelPolicy[] = rowsB.map(
    ({ environmentId: _e, ...p }) => p as SentinelPolicy,
  );

  const { merged, hasPending, actions, save, discard } = useSentinelDraft({
    envAId,
    envBId,
    policiesA,
    policiesB,
  });

  const handleAddPolicy = useCallback(() => setIsAddPanelOpen(true), []);

  const handleAdd = useCallback(
    (prodPolicy: SentinelPolicy | null, previewPolicy: SentinelPolicy | null) => {
      if (prodPolicy !== null && envAId) {
        collection.sentinelPolicies.insert({ ...prodPolicy, environmentId: envAId });
      }
      if (previewPolicy !== null && envBId) {
        collection.sentinelPolicies.insert({ ...previewPolicy, environmentId: envBId });
      }
    },
    [envAId, envBId],
  );

  const handleDelete = useCallback(
    (id: string) => {
      const keyA = `${envAId}::${id}`;
      const keyB = `${envBId}::${id}`;
      if (collection.sentinelPolicies.get(keyA)) {
        collection.sentinelPolicies.delete(keyA);
      }
      if (collection.sentinelPolicies.get(keyB)) {
        collection.sentinelPolicies.delete(keyB);
      }
    },
    [envAId, envBId],
  );

  return (
    <div className="flex flex-col gap-5">
      <SentinelPoliciesHeader onAddPolicy={handleAddPolicy} />
      {merged.length === 0 ? (
        <SentinelPoliciesEmpty />
      ) : (
        <SentinelPoliciesList
          envASlug={envA?.slug ?? "production"}
          envBSlug={envB?.slug ?? "preview"}
          merged={merged}
          actions={actions}
          onDelete={handleDelete}
        />
      )}
      <SentinelPoliciesUnsavedBar hasPending={hasPending} onSave={save} onDiscard={discard} />
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
