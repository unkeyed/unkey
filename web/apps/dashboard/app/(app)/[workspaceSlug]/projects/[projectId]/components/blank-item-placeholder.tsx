"use client";

import { useProjectItems } from "@/hooks/use-project-items";
import type { ProjectItemType } from "@/lib/project-items";
import { Cube, Database, Envelope, Lock } from "@unkey/icons";

type Props = {
  projectId: string;
  type: ProjectItemType;
  slug: string;
};

/**
 * Prototype placeholder rendered inside a project's typed item route
 * (apps/databases/queues/vault). Looks the item up in the localStorage-
 * backed list purely to show its display name; the rest of the page is
 * intentionally blank so the nav shape is the only thing under test.
 */
export function BlankItemPlaceholder({ projectId, type, slug }: Props) {
  const { items } = useProjectItems(projectId);
  const item = items.find((i) => i.type === type && i.slug === slug);
  const name = item?.name ?? slug;
  const Icon = iconForType(type);

  return (
    <div className="flex flex-col gap-2 p-6 w-full max-w-[1200px] mx-auto">
      <div className="flex items-center gap-3">
        <div className="size-10 bg-gray-3 rounded-[10px] flex items-center justify-center shrink-0 shadow-sm shadow-grayA-8/20 dark:ring-1 dark:ring-gray-4 dark:shadow-none">
          <Icon iconSize="xl-medium" className="size-5" />
        </div>
        <h1 className="text-xl font-semibold text-accent-12">{name}</h1>
      </div>
      <p className="text-sm text-gray-11">
        Placeholder page for this {type}. Content will land once the backend exposes it.
      </p>
    </div>
  );
}

function iconForType(type: ProjectItemType) {
  switch (type) {
    case "app":
      return Cube;
    case "database":
      return Database;
    case "queue":
      return Envelope;
    case "vault":
      return Lock;
  }
}
