"use client";

import { type SessionGroup, summarizeSessionGroup } from "@/lib/portal/sessions";
import { usePortalSessions } from "@/lib/portal/use-portal-sessions";
import { getErrorMessage } from "@/lib/unkey-client";
import { useWorkspace } from "@/providers/workspace-provider";
import type { V2PortalListSessionsSession } from "@unkey/api/models/components";
import { IconChevronRightOutline12, IconMagnifierOutline12 } from "@unkey/icons";
import {
  Badge,
  Button,
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  ResourceListBody,
  ResourceListContent,
  ResourceListFooter,
  ResourceListItem,
  Skeleton,
  TimestampInfo,
} from "@unkey/ui";
import { cn } from "cn";
import { useEffect, useId, useState } from "react";
import { RevokeSessionsDialog } from "./revoke-sessions-dialog";

const SEARCH_DEBOUNCE_MS = 300;
const SKELETON_ROWS = 3;

const STATUS_LABEL: Record<V2PortalListSessionsSession["status"], string> = {
  pending: "Link not opened",
  active: "Signed in",
};

export function PortalSessions({ portalId }: { portalId: string }) {
  const { user } = useWorkspace();
  const canRevoke = user?.role === "admin";

  const [searchInput, setSearchInput] = useState("");
  const [search, setSearch] = useState("");
  useEffect(() => {
    const timeout = setTimeout(() => setSearch(searchInput), SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(timeout);
  }, [searchInput]);

  const [revoking, setRevoking] = useState<SessionGroup | null>(null);
  const query = usePortalSessions(portalId, search);

  return (
    <section className="flex flex-col gap-3" aria-labelledby="portal-sessions-title">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="space-y-1">
          <h2 id="portal-sessions-title" className="text-sm font-medium text-gray-12">
            Sessions
          </h2>
          <p className="text-sm text-gray-11">
            Users with a portal session that has not expired or been revoked.
            {canRevoke ? null : " Only workspace admins can revoke sessions."}
          </p>
        </div>
        <label className="flex w-full items-center gap-2 rounded-lg border px-2.5 py-1.5 sm:w-64">
          <IconMagnifierOutline12 className="shrink-0 text-gray-9" />
          <input
            value={searchInput}
            onChange={(event) => setSearchInput(event.target.value)}
            maxLength={256}
            placeholder="Search by external ID"
            aria-label="Search sessions by external ID"
            className="w-full bg-transparent text-sm text-gray-12 placeholder:text-gray-9 focus:outline-hidden"
          />
        </label>
      </div>

      <SessionList query={query} search={search} canRevoke={canRevoke} onRevoke={setRevoking} />

      <RevokeSessionsDialog
        portalId={portalId}
        group={revoking}
        onOpenChange={(open) => {
          if (!open) {
            setRevoking(null);
          }
        }}
      />
    </section>
  );
}

function SessionList({
  query,
  search,
  canRevoke,
  onRevoke,
}: {
  query: ReturnType<typeof usePortalSessions>;
  search: string;
  canRevoke: boolean;
  onRevoke: (group: SessionGroup) => void;
}) {
  if (query.isLoading || (query.isPreviousData && query.groups.length === 0)) {
    return <SessionListSkeleton />;
  }

  if (query.isError && query.groups.length === 0) {
    return (
      <ResourceListContent>
        <div className="flex flex-col items-center gap-3 px-4 py-12 text-center">
          <span role="alert" className="text-sm text-gray-11">
            {getErrorMessage(query.error, "We couldn't load sessions.")}
          </span>
          <Button variant="outline" onClick={() => query.refetch()}>
            Retry
          </Button>
        </div>
      </ResourceListContent>
    );
  }

  // Groups can lapse between the API's two reads, so a page can come back empty
  // while more remain.
  if (query.groups.length === 0 && !query.hasNextPage) {
    return (
      <EmptyState>
        <EmptyStateHeader>
          <EmptyStateTitle>{search ? "No matching users" : "No active sessions"}</EmptyStateTitle>
          <EmptyStateDescription>
            {search
              ? `No user with an active session has an external ID starting with "${search}".`
              : "Users appear here once your app creates a portal session for them."}
          </EmptyStateDescription>
        </EmptyStateHeader>
      </EmptyState>
    );
  }

  return (
    <ResourceListContent aria-live="polite" className={cn(query.isPreviousData && "opacity-60")}>
      <ResourceListBody aria-label="Portal sessions">
        {query.groups.map((group) => (
          <SessionGroupRow
            key={group.externalId}
            group={group}
            canRevoke={canRevoke}
            onRevoke={() => onRevoke(group)}
          />
        ))}
      </ResourceListBody>
      {query.hasNextPage || (query.isError && query.groups.length > 0) ? (
        <ResourceListFooter className="justify-center gap-3">
          {query.isError ? (
            <span role="alert" className="text-sm text-gray-11">
              We couldn't load sessions.
            </span>
          ) : null}
          <Button
            variant="outline"
            loading={query.isFetchingNextPage}
            disabled={query.isFetchingNextPage}
            onClick={() => (query.hasNextPage ? query.fetchNextPage() : query.refetch())}
          >
            {query.isError ? "Retry" : "Load more"}
          </Button>
        </ResourceListFooter>
      ) : null}
    </ResourceListContent>
  );
}

function SessionGroupRow({
  group,
  canRevoke,
  onRevoke,
}: {
  group: SessionGroup;
  canRevoke: boolean;
  onRevoke: () => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const panelId = useId();
  const summary = summarizeSessionGroup(group);

  return (
    <ResourceListItem>
      <div className="flex flex-wrap items-center gap-3 px-4 py-3">
        <button
          type="button"
          aria-expanded={expanded}
          aria-controls={panelId}
          onClick={() => setExpanded((open) => !open)}
          className="flex min-w-0 flex-1 items-center gap-2 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7"
        >
          <IconChevronRightOutline12
            className={cn("shrink-0 text-gray-9 transition-transform", expanded && "rotate-90")}
          />
          <span className="truncate font-mono text-sm text-gray-12">{group.externalId}</span>
          <Badge variant="secondary" size="sm">
            {summary.count} {summary.count === 1 ? "session" : "sessions"}
          </Badge>
        </button>
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-9">
          <span className="flex items-center gap-1">
            Latest <TimestampInfo value={summary.lastCreatedAt} displayType="relative" />
          </span>
          <span className="flex items-center gap-1">
            Expires <TimestampInfo value={summary.lastExpiresAt} displayType="relative" />
          </span>
        </div>
        {canRevoke ? (
          <Button variant="outline" color="danger" size="sm" onClick={onRevoke}>
            Revoke
          </Button>
        ) : null}
      </div>
      {expanded ? (
        <ul id={panelId} className="flex flex-col gap-2 border-t bg-grayA-2 px-4 py-3 pl-9">
          {group.sessions.map((session) => (
            <li key={session.id} className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs">
              <Badge variant={session.status === "active" ? "success" : "warning"} size="sm">
                {STATUS_LABEL[session.status]}
              </Badge>
              <span className="flex items-center gap-1 text-gray-9">
                Created <TimestampInfo value={session.createdAt} displayType="relative" />
              </span>
              <span className="flex items-center gap-1 text-gray-9">
                Expires <TimestampInfo value={session.expiresAt} displayType="relative" />
              </span>
              <span className="flex flex-wrap gap-1">
                {session.scopes.map((scope) => (
                  <Badge key={scope} variant="primary" size="sm" font="mono">
                    {scope}
                  </Badge>
                ))}
              </span>
            </li>
          ))}
        </ul>
      ) : null}
    </ResourceListItem>
  );
}

function SessionListSkeleton() {
  return (
    <ResourceListContent aria-busy="true" aria-live="polite">
      <output className="sr-only">Loading sessions…</output>
      <ResourceListBody aria-hidden="true">
        {Array.from({ length: SKELETON_ROWS }).map((_, index) => (
          <ResourceListItem
            // biome-ignore lint/suspicious/noArrayIndexKey: skeleton rows are static and never reorder
            key={index}
            className="flex items-center gap-3 px-4 py-3"
          >
            <Skeleton className="h-3.5 w-40" />
            <Skeleton className="h-5 w-20 rounded-md" />
            <Skeleton className="ml-auto h-3 w-32" />
          </ResourceListItem>
        ))}
      </ResourceListBody>
    </ResourceListContent>
  );
}
