"use client";

import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { EmptyRootKeys, ResourceListBody, ResourceListContent, Skeleton } from "@unkey/ui";
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
          <EmptyRootKeys />
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
