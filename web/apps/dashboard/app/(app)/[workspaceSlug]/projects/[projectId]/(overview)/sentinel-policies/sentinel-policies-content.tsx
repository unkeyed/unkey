"use client";

import { collection } from "@/lib/collections";
import type { SentinelPolicy } from "@/lib/collections/deploy/sentinel-policies.schema";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useCallback, useState } from "react";
import { useProjectData } from "../data-provider";
import { useOptionalProjectLayout } from "../layout-provider";
import { SentinelPolicyPanel } from "./components/add-panel";
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
  const [editing, setEditing] = useState<SentinelPolicy | null>(null);

  const { data: rowsA } = useLiveQuery(
    (q) =>
      q
        .from({ p: collection.sentinelPolicies })
        .where(({ p }) => eq(p.environmentId, envAId))
        .orderBy(({ p }) => p._order),
    [envAId],
  );

  const { data: rowsB } = useLiveQuery(
    (q) =>
      q
        .from({ p: collection.sentinelPolicies })
        .where(({ p }) => eq(p.environmentId, envBId))
        .orderBy(({ p }) => p._order),
    [envBId],
  );

  // Strip the row-only `environmentId` field — downstream draft logic operates
  // on bare SentinelPolicy values.
  const policiesA: SentinelPolicy[] = rowsA.map(
    ({ environmentId: _e, _order: _o, ...p }) => p as SentinelPolicy,
  );
  const policiesB: SentinelPolicy[] = rowsB.map(
    ({ environmentId: _e, _order: _o, ...p }) => p as SentinelPolicy,
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

  const envASlug = envA?.slug ?? "production";
  const envBSlug = envB?.slug ?? "preview";

  const handleEdit = useCallback((policy: SentinelPolicy) => {
    setEditing(policy);
  }, []);

  const handleSave = useCallback(
    (updated: SentinelPolicy) => {
      for (const envId of [envAId, envBId]) {
        if (!envId) { continue };
        const key = `${envId}::${updated.id}`;
        if (!collection.sentinelPolicies.get(key)) { continue };
        collection.sentinelPolicies.update(key, (draft) => {
          Object.assign(draft, updated, { environmentId: envId });
        });
      }
      setEditing(null);
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
          envASlug={envASlug}
          envBSlug={envBSlug}
          merged={merged}
          actions={actions}
          onDelete={handleDelete}
          onEdit={handleEdit}
        />
      )}
      <SentinelPoliciesUnsavedBar hasPending={hasPending} onSave={save} onDiscard={discard} />
      <SentinelPolicyPanel
        mode="add"
        envASlug={envASlug}
        envBSlug={envBSlug}
        isOpen={isAddPanelOpen}
        topOffset={layout?.tableDistanceToTop ?? 0}
        onClose={() => setIsAddPanelOpen(false)}
        onAdd={handleAdd}
      />
      {editing !== null && (
        <SentinelPolicyPanel
          key={editing.id}
          mode="edit"
          envASlug={envASlug}
          envBSlug={envBSlug}
          isOpen={true}
          topOffset={layout?.tableDistanceToTop ?? 0}
          onClose={() => setEditing(null)}
          initialPolicy={editing}
          onSave={handleSave}
        />
      )}
    </div>
  );
}
