"use client";

import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { IconBookBookmarkOutline18 } from "@unkey/icons";
import {
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  ResourceListBody,
  ResourceListContent,
  Skeleton,
  buttonVariants,
} from "@unkey/ui";
import { RootKeyRow } from "./root-key-row";

type RootKeysResourceListProps = {
  rootKeys: RootKey[];
  isLoading: boolean;
  onSelect: (rootKey: RootKey) => void;
  onEditKey: (rootKey: RootKey) => void;
};

export function RootKeysResourceList({
  rootKeys,
  isLoading,
  onSelect,
  onEditKey,
}: RootKeysResourceListProps) {
  if (isLoading) {
    return (
      <ResourceListContent>
        <ResourceListBody>
          {Array.from({ length: 5 }, (_, index) => index).map((index) => (
            <li key={index} className="flex items-center gap-4 px-4 py-3">
              <Skeleton className="h-8 flex-1" />
            </li>
          ))}
        </ResourceListBody>
      </ResourceListContent>
    );
  }

  if (rootKeys.length === 0) {
    return (
      <ResourceListContent>
        <div className="flex w-full items-center justify-center px-4 py-16">
          <EmptyState frame="none">
            <EmptyStateHeader>
              <EmptyStateTitle>No Root Keys Found</EmptyStateTitle>
              <EmptyStateDescription>
                There are no root keys configured yet. Create your first root key to start managing
                permissions and access control.
              </EmptyStateDescription>
            </EmptyStateHeader>
            <EmptyStateActions>
              <a
                href="https://www.unkey.com/docs/security/overview#root-keys"
                target="_blank"
                rel="noopener noreferrer"
                className={buttonVariants({ variant: "outline", size: "md" })}
              >
                <IconBookBookmarkOutline18 />
                Learn about Root Keys
              </a>
            </EmptyStateActions>
          </EmptyState>
        </div>
      </ResourceListContent>
    );
  }

  return (
    <ResourceListContent>
      <ResourceListBody>
        {rootKeys.map((rootKey) => (
          <RootKeyRow
            key={rootKey.id}
            rootKey={rootKey}
            onSelect={onSelect}
            onEditKey={onEditKey}
          />
        ))}
      </ResourceListBody>
    </ResourceListContent>
  );
}
