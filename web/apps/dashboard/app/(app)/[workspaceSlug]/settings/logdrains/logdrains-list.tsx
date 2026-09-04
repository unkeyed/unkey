"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { CloudUp, Database, Earth, Layers3, ShareUpRight } from "@unkey/icons";
import {
  Button,
  EmptyHero,
  InfoTooltip,
  ResourceListBody,
  ResourceListContent,
  ResourceListItem,
  Skeleton,
} from "@unkey/ui";
import { formatDistanceToNow } from "date-fns";
import Link from "next/link";
import { CreateLogdrainButton } from "./create-logdrain-button";
import { DrainMedia } from "./drain-destinations";
import type { DrainListItem } from "./drain-schema";
import { DrainStatusBadge } from "./drain-status-badge";

const SKELETON_ROWS = 5;

function DrainRow({ drain, workspaceSlug }: { drain: DrainListItem; workspaceSlug: string }) {
  return (
    <ResourceListItem>
      <Link
        href={routes.settings.logdrains.detail({ workspaceSlug, drainId: drain.id })}
        aria-label={`Open ${drain.name}`}
        className="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-grayA-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7"
      >
        <DrainMedia kind={drain.kind} />

        <div className="flex min-w-0 flex-1 items-center gap-2">
          <InfoTooltip content={drain.name} asChild position={{ align: "start", side: "top" }}>
            <span className="truncate text-[13px] font-medium text-accent-12">{drain.name}</span>
          </InfoTooltip>
          <span className="shrink-0">
            <DrainStatusBadge status={drain.status} />
          </span>
        </div>

        {/* A plain string, not TimestampInfo: its popover trigger is a button and this row is
            already a link. Same wording as TimestampInfo's relative display. */}
        <span className="shrink-0 text-xs text-gray-9">
          {formatDistanceToNow(new Date(drain.createdAt), { addSuffix: true })}
        </span>
      </Link>
    </ResourceListItem>
  );
}

function DrainListSkeleton() {
  return (
    <ResourceListContent aria-busy="true" aria-live="polite">
      <output className="sr-only">Loading log drains…</output>
      <ResourceListBody aria-hidden="true">
        {Array.from({ length: SKELETON_ROWS }).map((_, index) => (
          <ResourceListItem
            // biome-ignore lint/suspicious/noArrayIndexKey: skeleton rows are static and never reorder
            key={index}
            className="flex items-center gap-3 px-4 py-3"
          >
            <Skeleton className="size-8 rounded-[10px]" />
            <Skeleton className="h-3.5 w-40" />
            <Skeleton className="h-5 w-20 rounded-md" />
            <Skeleton className="ml-auto h-3 w-24" />
          </ResourceListItem>
        ))}
      </ResourceListBody>
    </ResourceListContent>
  );
}

export function LogdrainsList({ onCreate }: { onCreate: () => void }) {
  const workspace = useWorkspaceNavigation();
  const query = trpc.logdrain.list.useQuery();

  if (query.isLoading) {
    return <DrainListSkeleton />;
  }

  if (query.isError) {
    return (
      <ResourceListContent>
        <div className="flex flex-col items-center gap-3 px-4 py-16 text-center">
          <span role="alert" className="text-sm text-gray-11">
            We couldn't load log drains.
          </span>
          <Button variant="outline" onClick={() => query.refetch()}>
            Retry
          </Button>
        </div>
      </ResourceListContent>
    );
  }

  if (!query.data?.length) {
    return (
      <EmptyHero>
        <EmptyHero.Icons>
          <Layers3 iconSize="md-medium" />
          <ShareUpRight iconSize="md-medium" />
          <CloudUp iconSize="md-thin" />
          <Earth iconSize="md-medium" />
          <Database iconSize="md-medium" />
        </EmptyHero.Icons>
        <EmptyHero.Title>Create your first log drain</EmptyHero.Title>
        <EmptyHero.Description>
          Send audit logs to an HTTPS endpoint or an Axiom dataset.
        </EmptyHero.Description>
        <EmptyHero.Actions>
          <CreateLogdrainButton onClick={onCreate} />
        </EmptyHero.Actions>
      </EmptyHero>
    );
  }

  return (
    <ResourceListContent aria-live="polite">
      <ResourceListBody aria-label="Log drains">
        {query.data.map((drain) => (
          <DrainRow key={drain.id} drain={drain} workspaceSlug={workspace.slug} />
        ))}
      </ResourceListBody>
    </ResourceListContent>
  );
}
