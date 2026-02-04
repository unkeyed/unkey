"use client";
import { Link4, Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
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
    <div
      className={cn(
        "border border-gray-4 rounded-[14px] overflow-hidden",
        customDomains.length === 0 && !isAddingNew && !isLoading && "border-dashed",
      )}
    >
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
    <div className="px-4 py-8 flex justify-center items-center min-h-[150px] relative group">
      <div className="flex flex-col items-center gap-3 text-center">
        {/* Icon with subtle animation */}
        <div className="relative">
          <div className="absolute inset-0 bg-gradient-to-r from-accent-4 to-accent-3 rounded-full blur-xl opacity-20 group-hover:opacity-30 transition-opacity duration-300 animate-pulse" />
          <div className="relative bg-gray-3 rounded-full p-3 group-hover:bg-gray-4 transition-all duration-200">
            <Link4
              className="text-gray-9 size-6 group-hover:text-gray-11 transition-all duration-200 animate-pulse"
              style={{ animationDuration: "2s" }}
            />
          </div>
        </div>
        {/* Content */}
        <div className="space-y-2">
          <h3 className="text-gray-12 font-medium text-sm">No custom domains configured</h3>
          {hasEnvironments ? (
            <p className="text-gray-9 text-xs max-w-[280px] leading-relaxed">
              Add a custom domain to serve your application from your own domain.
            </p>
          ) : (
            <p className="text-gray-8 text-xs max-w-[280px] leading-relaxed">
              Create an environment first to add custom domains
            </p>
          )}
        </div>
        {/* Button */}
        {hasEnvironments && (
          <Button size="sm" variant="primary" onClick={onAdd} className="gap-1.5 mt-1">
            <Plus className="!size-3" />
            Add domain
          </Button>
        )}
      </div>
    </div>
  );
}
