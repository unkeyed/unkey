"use client";

import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { useCallback, useState } from "react";
import { useProjectData } from "../data-provider";
import { SentinelPoliciesEmpty } from "./components/sentinel-policies-empty";
import { SentinelPoliciesHeader } from "./components/sentinel-policies-header";
import { SentinelPoliciesList } from "./components/sentinel-policies-list";
import { SentinelPoliciesToolbar } from "./components/sentinel-policies-toolbar";

export function SentinelPoliciesContent() {
  const { environments } = useProjectData();

  const defaultEnvId =
    environments.find((e) => e.slug === "production")?.id ?? environments.at(0)?.id ?? "";

  const [environmentId, setEnvironmentId] = useState(defaultEnvId);

  const { data } = useLiveQuery(
    (q) =>
      q
        .from({ s: collection.environmentSettings })
        .where(({ s }) => eq(s.environmentId, environmentId)),
    [environmentId],
  );

  const settings = data.at(0);
  const policies = settings?.sentinelConfig?.policies ?? [];

  const handleAddPolicy = useCallback(() => {
    // Placeholder — policy creation forms will be added later
  }, []);

  return (
    <div className="flex flex-col gap-5">
      <SentinelPoliciesHeader onAddPolicy={handleAddPolicy} />
      <SentinelPoliciesToolbar
        environmentId={environmentId}
        onEnvironmentChange={setEnvironmentId}
        environments={environments}
      />
      {policies.length === 0 ? (
        <SentinelPoliciesEmpty />
      ) : (
        <SentinelPoliciesList environmentId={environmentId} policies={policies} />
      )}
    </div>
  );
}
