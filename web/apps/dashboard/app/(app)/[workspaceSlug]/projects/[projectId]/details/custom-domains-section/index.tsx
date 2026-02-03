"use client";
import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useState } from "react";
import { AddCustomDomain } from "./add-custom-domain";
import { CustomDomainRow, CustomDomainRowSkeleton } from "./custom-domain-row";
import { useCustomDomainsManager } from "./hooks/use-custom-domains-manager";

type CustomDomainsSectionProps = {
  projectId: string;
  environments: Array<{ id: string; slug: string }>;
};

export function CustomDomainsSection({ projectId, environments }: CustomDomainsSectionProps) {
  const { customDomains, isLoading, getExistingDomain, invalidate } = useCustomDomainsManager({
    projectId,
  });
  const [isAddingNew, setIsAddingNew] = useState(false);

  const startAdding = () => setIsAddingNew(true);
  const cancelAdding = () => setIsAddingNew(false);

  return (
    <div className="border border-gray-4 rounded-lg overflow-hidden">
      {/* Domain list */}
      <div className="divide-y divide-gray-4">
        {isLoading ? (
          <>
            <CustomDomainRowSkeleton />
            <CustomDomainRowSkeleton />
          </>
        ) : (
          customDomains.map((domain) => (
            <CustomDomainRow
              key={domain.id}
              domain={domain}
              projectId={projectId}
              onDelete={invalidate}
              onRetry={invalidate}
            />
          ))
        )}

        {isAddingNew && (
          <AddCustomDomain
            projectId={projectId}
            environments={environments}
            getExistingDomain={getExistingDomain}
            onCancel={cancelAdding}
            onSuccess={() => {
              invalidate();
              cancelAdding();
            }}
          />
        )}

        {customDomains.length === 0 && !isAddingNew && !isLoading && (
          <EmptyState onAdd={startAdding} hasEnvironments={environments.length > 0} />
        )}
      </div>

      {/* Add button footer - only show when not adding and has domains */}
      {!isAddingNew && customDomains.length > 0 && environments.length > 0 && (
        <div className="px-4 py-2 border-t border-gray-4 bg-gray-2">
          <Button size="sm" variant="ghost" onClick={startAdding} className="text-gray-9 gap-1.5">
            <Plus className="!size-3" />
            Add domain
          </Button>
        </div>
      )}
    </div>
  );
}

function EmptyState({ onAdd, hasEnvironments }: { onAdd: () => void; hasEnvironments: boolean }) {
  return (
    <div className="px-4 py-8 text-center flex flex-col items-center gap-3">
      <p className="text-gray-9 text-sm">No custom domains configured</p>
      {hasEnvironments ? (
        <Button size="sm" variant="outline" onClick={onAdd} className="gap-1.5">
          <Plus className="!size-3" />
          Add domain
        </Button>
      ) : (
        <p className="text-gray-8 text-xs">Create an environment first to add custom domains</p>
      )}
    </div>
  );
}
