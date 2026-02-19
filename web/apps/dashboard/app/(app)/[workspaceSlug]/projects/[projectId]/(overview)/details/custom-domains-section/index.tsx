"use client";
import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useState } from "react";
import { EmptySection } from "../../components/empty-section";
import { useProjectData } from "../../data-provider";
import { CustomDomainRow } from "../../settings/components/advanced-settings/custom-domains/custom-domain-row";
import { AddCustomDomain } from "./add-custom-domain";

type CustomDomainsSectionProps = {
  environments: Array<{ id: string; slug: string }>;
};

export function CustomDomainsSection({ environments }: CustomDomainsSectionProps) {
  const { customDomains, isCustomDomainsLoading } = useProjectData();
  const [isAddingNew, setIsAddingNew] = useState(false);

  const getExistingDomain = (domain: string) =>
    customDomains.find((d) => d.domain.toLowerCase() === domain.toLowerCase());

  const startAdding = () => setIsAddingNew(true);
  const cancelAdding = () => setIsAddingNew(false);

  return (
    <div
      className={cn(
        "border border-gray-4 rounded-[14px] overflow-hidden",
        customDomains.length === 0 && !isAddingNew && !isCustomDomainsLoading && "border-dashed",
      )}
    >
      {/* Domain list */}
      <div className="divide-y divide-gray-4">
        {isCustomDomainsLoading ? (
          <>
            <CustomDomainRowSkeleton />
            <CustomDomainRowSkeleton />
          </>
        ) : (
          customDomains.map((domain) => <CustomDomainRow key={domain.id} domain={domain} />)
        )}

        {isAddingNew && (
          <AddCustomDomain
            environments={environments}
            getExistingDomain={getExistingDomain}
            onDismiss={cancelAdding}
          />
        )}

        {customDomains.length === 0 && !isAddingNew && !isCustomDomainsLoading && (
          <EmptyState onAdd={startAdding} hasEnvironments={environments.length > 0} />
        )}
      </div>

      {/* Add button footer - only show when not adding and has domains */}
      {!isAddingNew && customDomains.length > 0 && environments.length > 0 && (
        <div className="px-4 py-2 border-t border-gray-4 bg-gray-2">
          <Button size="sm" variant="ghost" onClick={startAdding} className="text-gray-9 gap-1.5">
            <Plus className="size-3!" />
            Add domain
          </Button>
        </div>
      )}
    </div>
  );
}

function EmptyState({ onAdd, hasEnvironments }: { onAdd: () => void; hasEnvironments: boolean }) {
  return (
    <EmptySection
      title="No custom domains configured"
      description={
        hasEnvironments
          ? "Add a custom domain to serve your application from your own domain."
          : "Create an environment first to add custom domains"
      }
    >
      {hasEnvironments && (
        <Button size="sm" variant="primary" onClick={onAdd} className="gap-1.5 mt-1">
          <Plus className="size-3!" />
          Add domain
        </Button>
      )}
    </EmptySection>
  );
}

export function CustomDomainRowSkeleton() {
  return (
    <div className="flex items-center justify-between px-4 py-3 border-b border-gray-4 last:border-b-0">
      <div className="flex items-center gap-3">
        <div className="w-4 h-4 bg-gray-4 rounded animate-pulse" />
        <div className="w-32 h-4 bg-gray-4 rounded animate-pulse" />
      </div>
      <div className="w-16 h-5 bg-gray-4 rounded animate-pulse" />
    </div>
  );
}
