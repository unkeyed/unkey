"use client";

import { collection } from "@/lib/collections";
import type { SentinelPolicy } from "@/lib/trpc/routers/deploy/environment-settings/sentinel/update-middleware";
import { Reorder } from "framer-motion";
import { useCallback, useEffect, useState } from "react";
import { SentinelPolicyRow } from "./sentinel-policy-row";

type SentinelPoliciesListProps = {
  environmentId: string;
  policies: SentinelPolicy[];
};

export function SentinelPoliciesList({ environmentId, policies }: SentinelPoliciesListProps) {
  const [orderedPolicies, setOrderedPolicies] = useState(policies);

  useEffect(() => {
    setOrderedPolicies(policies);
  }, [policies]);

  const persist = useCallback(
    (updated: SentinelPolicy[]) => {
      collection.environmentSettings.update(environmentId, (draft) => {
        draft.sentinelConfig = { policies: updated };
      });
    },
    [environmentId],
  );

  const handleReorder = useCallback(
    (newOrder: SentinelPolicy[]) => {
      setOrderedPolicies(newOrder);
      persist(newOrder);
    },
    [persist],
  );

  const handleToggleActive = useCallback(
    (id: string) => {
      setOrderedPolicies((prev) => {
        const next = prev.map((p) => (p.id === id ? { ...p, enabled: !p.enabled } : p));
        persist(next);
        return next;
      });
    },
    [persist],
  );

  const handleUpdate = useCallback(
    (id: string, field: "name", value: string) => {
      setOrderedPolicies((prev) => {
        const next = prev.map((p) => (p.id === id ? { ...p, [field]: value } : p));
        persist(next);
        return next;
      });
    },
    [persist],
  );

  return (
    <div className="border border-grayA-4 rounded-[14px] overflow-hidden">
      <Reorder.Group axis="y" values={orderedPolicies} onReorder={handleReorder} as="div">
        {orderedPolicies.map((policy, i) => (
          <SentinelPolicyRow
            key={policy.id}
            policy={policy}
            index={i}
            isLast={i === orderedPolicies.length - 1}
            onToggleActive={handleToggleActive}
            onUpdate={handleUpdate}
          />
        ))}
      </Reorder.Group>
    </div>
  );
}
