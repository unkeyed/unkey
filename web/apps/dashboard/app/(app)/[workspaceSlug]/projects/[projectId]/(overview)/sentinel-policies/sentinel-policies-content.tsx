"use client";

import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useCallback, useMemo } from "react";
import { useProjectData } from "../data-provider";
import { SentinelPoliciesEmpty } from "./components/sentinel-policies-empty";
import { SentinelPoliciesHeader } from "./components/sentinel-policies-header";
import { SentinelPoliciesList } from "./components/sentinel-policies-list";
import { SentinelPoliciesToolbar } from "./components/sentinel-policies-toolbar";

export function SentinelPoliciesContent() {
  const { environments } = useProjectData();

  const envA = environments.find((e) => e.slug === "production") ?? environments.at(0);
  const envB = environments.find((e) => e.id !== envA?.id) ?? environments.at(1);

  const envAId = envA?.id ?? "";
  const envBId = envB?.id ?? "";

  const { data: dataA } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) => eq(s.environmentId, envAId)),
    [envAId],
  );

  const { data: dataB } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) => eq(s.environmentId, envBId)),
    [envBId],
  );

  const policiesA = dataA.at(0)?.sentinelConfig?.policies ?? [];
  const policiesB = dataB.at(0)?.sentinelConfig?.policies ?? [];

  const policyCount = useMemo(() => {
    const ids = new Set([...policiesA.map((p) => p.id), ...policiesB.map((p) => p.id)]);
    return ids.size;
  }, [policiesA, policiesB]);

  const handleAddPolicy = useCallback(() => {
    // Placeholder — policy creation forms will be added later
  }, []);

  return (
    <div className="flex flex-col gap-5">
      <SentinelPoliciesHeader onAddPolicy={handleAddPolicy} />
      <SentinelPoliciesToolbar
        policyCount={policyCount}
        envASlug={envA?.slug ?? "production"}
        envBSlug={envB?.slug ?? "preview"}
      />
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
        />
      )}
    </div>
  );
}
