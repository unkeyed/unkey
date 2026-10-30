"use client";

import { ListPager } from "@/components/list-pager";
import { LoadError } from "@/components/load-error";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { useInfiniteList } from "@/lib/loadable/list";
import {
  type SessionGroup,
  sessionGroupScopes,
  summarizeSessionGroup,
} from "@/lib/portal/sessions";
import { usePortalSessions } from "@/lib/portal/use-portal-sessions";
import type { Scope, V2PortalListSessionsSession } from "@unkey/api/models/components";
import {
  IconBanOutline18,
  IconChartActivity2Outline18,
  IconDotsOutline18,
  IconEyeOutline18,
  IconKey2Outline18,
  IconRefresh3Outline18,
  IconShieldKeyOutline18,
  IconUserOutline12,
} from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Button,
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateTitle,
  ResourceList,
  ResourceListBody,
  ResourceListContent,
  ResourceListHeader,
  ResourceListItem,
  ResourceListSearch,
  SearchInput,
  Skeleton,
  SlidePanel,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelFooter,
  SlidePanelHeader,
  SlidePanelTitle,
  TimestampInfo,
} from "@unkey/ui";
import { cn } from "cn";
import { type ComponentType, useEffect, useState } from "react";
import { RevokeSessionsDialog } from "./revoke-sessions-dialog";

const SEARCH_DEBOUNCE_MS = 300;
const SKELETON_KEYS = ["skeleton-1", "skeleton-2", "skeleton-3"];

const GRID =
  "grid grid-cols-[minmax(0,2fr)_minmax(0,0.6fr)_minmax(0,1fr)_minmax(0,1fr)_32px] items-center gap-4 px-4";

const PANEL_GRID = "grid grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)_minmax(0,1fr)] gap-4 px-4";

type ScopeDisplay = { label: string; Icon: ComponentType<{ className?: string }> };

const SCOPES = new Map<string, ScopeDisplay>(
  Object.entries({
    "keys:read": { label: "View keys", Icon: IconKey2Outline18 },
    "keys:reroll": { label: "Reroll keys", Icon: IconRefresh3Outline18 },
    "analytics:read": { label: "View analytics", Icon: IconChartActivity2Outline18 },
  } satisfies Record<Scope, ScopeDisplay>),
);

const STATUS_LABEL: Record<V2PortalListSessionsSession["status"], string> = {
  pending: "Link not opened",
  active: "Signed in",
};

export function PortalSessions({ portalId, canRevoke }: { portalId: string; canRevoke: boolean }) {
  const [searchInput, setSearchInput] = useState("");
  const [search, setSearch] = useState("");
  useEffect(() => {
    const timeout = setTimeout(() => setSearch(searchInput), SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(timeout);
  }, [searchInput]);

  const [revoking, setRevoking] = useState<SessionGroup | null>(null);
  const [panel, setPanel] = useState<{ group: SessionGroup; open: boolean } | null>(null);
  const query = usePortalSessions(portalId, search);
  const openGroup = panel?.open
    ? query.groups.find((group) => group.externalId === panel.group.externalId)
    : undefined;

  return (
    <ResourceList>
      <ResourceListHeader>
        <ResourceListSearch>
          <SearchInput
            value={searchInput}
            onValueChange={setSearchInput}
            label="Search sessions by external ID"
            placeholder="Search by external ID"
          />
        </ResourceListSearch>
      </ResourceListHeader>

      <SessionsBody
        query={query}
        search={search}
        selectedId={openGroup?.externalId}
        canRevoke={canRevoke}
        onOpen={(group) => setPanel({ group, open: true })}
        onRevoke={setRevoking}
      />

      <SessionGroupPanel
        group={openGroup ?? panel?.group ?? null}
        isOpen={openGroup !== undefined}
        canRevoke={canRevoke}
        onClose={() => setPanel((current) => current && { ...current, open: false })}
        onExitComplete={() => setPanel(null)}
        onRevoke={setRevoking}
      />

      <RevokeSessionsDialog
        portalId={portalId}
        group={revoking}
        onOpenChange={(open) => {
          if (!open) {
            setRevoking(null);
          }
        }}
      />
    </ResourceList>
  );
}

function SessionsBody({
  query,
  search,
  selectedId,
  canRevoke,
  onOpen,
  onRevoke,
}: {
  query: ReturnType<typeof usePortalSessions>;
  search: string;
  selectedId: string | undefined;
  canRevoke: boolean;
  onOpen: (group: SessionGroup) => void;
  onRevoke: (group: SessionGroup) => void;
}) {
  const list = useInfiniteList(query, (page) => page.groups);

  return match(list)
    .with({ status: "loading" }, () => <SessionsTableSkeleton />)
    .with({ status: "failed" }, ({ error, retry }) => (
      <ResourceListContent>
        <LoadError title="Could not load sessions" error={error} frame="none" onRetry={retry} />
      </ResourceListContent>
    ))
    .with({ status: "empty" }, () => <SessionsEmpty search={search} />)
    .with({ status: "ready" }, ({ rows, footer }) => (
      <ResourceListContent aria-live="polite" className={cn(query.isPreviousData && "opacity-60")}>
        <div className="overflow-x-auto">
          <div className="min-w-[640px]">
            <SessionsTableHeader />
            <ResourceListBody aria-label="Portal sessions">
              {rows.map((group) => (
                <SessionGroupRow
                  key={group.externalId}
                  group={group}
                  selected={group.externalId === selectedId}
                  canRevoke={canRevoke}
                  onOpen={() => onOpen(group)}
                  onRevoke={() => onRevoke(group)}
                />
              ))}
            </ResourceListBody>
          </div>
        </div>
        <ListPager state={footer} noun="sessions" />
      </ResourceListContent>
    ))
    .exhaustive();
}

function SessionsEmpty({ search }: { search: string }) {
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

function SessionsTableHeader() {
  return (
    <div className={cn(GRID, "border-b bg-table-header py-[7px] text-xs font-medium text-gray-12")}>
      <span>External ID</span>
      <span>Sessions</span>
      <span>Last created</span>
      <span>Expires</span>
      <span />
    </div>
  );
}

function SessionGroupRow({
  group,
  selected,
  canRevoke,
  onOpen,
  onRevoke,
}: {
  group: SessionGroup;
  selected: boolean;
  canRevoke: boolean;
  onOpen: () => void;
  onRevoke: () => void;
}) {
  const summary = summarizeSessionGroup(group);
  const actions: MenuItem[] = [
    {
      id: "view",
      label: "View sessions",
      icon: <IconEyeOutline18 className="size-3.5" />,
      onClick: onOpen,
      divider: canRevoke,
    },
    ...(canRevoke
      ? [
          {
            id: "revoke",
            label: "Revoke sessions",
            icon: <IconBanOutline18 className="size-3.5" />,
            onClick: onRevoke,
            className: "text-error-11 hover:bg-error-3 focus:bg-error-3",
          },
        ]
      : []),
  ];

  return (
    <ResourceListItem
      className={cn(
        GRID,
        "h-12 text-xs text-gray-12 transition-colors hover:bg-grayA-2",
        selected && "bg-grayA-3 hover:bg-grayA-3",
      )}
    >
      <button
        type="button"
        aria-label={`View sessions for ${group.externalId}`}
        onClick={onOpen}
        className="absolute inset-0 z-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-grayA-7"
      />
      <span className="flex min-w-0 items-center gap-2.5">
        <span className="flex size-6 shrink-0 items-center justify-center rounded-full border bg-gray-3">
          <IconUserOutline12 className="text-gray-11" />
        </span>
        <span data-sentry-mask className="truncate text-sm font-medium">
          {group.externalId}
        </span>
      </span>
      <span className="text-gray-11">{summary.count}</span>
      <span className="min-w-0 truncate text-gray-11">
        <span className="relative z-10 inline-flex">
          <TimestampInfo value={summary.lastCreatedAt} displayType="relative" />
        </span>
      </span>
      <span className="min-w-0 truncate text-gray-11">
        <span className="relative z-10 inline-flex">
          <TimestampInfo value={summary.lastExpiresAt} displayType="relative" />
        </span>
      </span>
      <span className="relative z-10 flex justify-end">
        <TableActionPopover items={actions}>
          <Button
            variant="ghost"
            size="icon"
            className="shrink-0"
            title="Session actions"
            aria-label={`Actions for ${group.externalId}`}
          >
            <IconDotsOutline18 />
          </Button>
        </TableActionPopover>
      </span>
    </ResourceListItem>
  );
}

function SessionGroupPanel({
  group,
  isOpen,
  canRevoke,
  onClose,
  onExitComplete,
  onRevoke,
}: {
  group: SessionGroup | null;
  isOpen: boolean;
  canRevoke: boolean;
  onClose: () => void;
  onExitComplete: () => void;
  onRevoke: (group: SessionGroup) => void;
}) {
  return (
    <SlidePanel
      isOpen={isOpen}
      onClose={onClose}
      onExitComplete={onExitComplete}
      widthClassName="w-140"
    >
      {group ? (
        <>
          <SlidePanelHeader className="items-center">
            <SlidePanelTitle data-sentry-mask className="min-w-0 truncate">
              {group.externalId}
            </SlidePanelTitle>
            <SlidePanelCloseButton />
          </SlidePanelHeader>
          <SlidePanelContent className="flex flex-col gap-6 overflow-y-auto px-6 py-4">
            <section className="flex flex-col gap-2">
              <h3 className="text-xs font-medium text-gray-12">Permissions</h3>
              <ul className="divide-y divide-grayA-4 rounded-lg border text-sm text-gray-11">
                {sessionGroupScopes(group).map((scope) => {
                  const { label, Icon } = SCOPES.get(scope) ?? {
                    label: scope,
                    Icon: IconShieldKeyOutline18,
                  };
                  return (
                    <li key={scope} className="flex items-center gap-3 px-4 py-2.5">
                      <Icon className="size-3.5 shrink-0 text-gray-11" />
                      <span className="flex-1 truncate font-medium text-gray-12">{label}</span>
                      <span className="text-xs text-gray-9">{scope}</span>
                    </li>
                  );
                })}
              </ul>
            </section>
            <section className="flex flex-col gap-2">
              <h3 className="text-xs font-medium text-gray-12">Sessions</h3>
              <div className="overflow-hidden rounded-lg border">
                <div
                  className={cn(
                    PANEL_GRID,
                    "border-b bg-table-header py-[7px] text-xs font-medium text-gray-12",
                  )}
                >
                  <span>Status</span>
                  <span>Created</span>
                  <span>Expires</span>
                </div>
                <ul className="divide-y divide-grayA-4">
                  {group.sessions.map((session) => (
                    <li
                      key={session.id}
                      className={cn(PANEL_GRID, "h-10 items-center text-xs text-gray-11")}
                    >
                      <span className="flex min-w-0 items-center gap-2 text-gray-12">
                        <span
                          className={cn(
                            "size-1.5 shrink-0 rounded-full",
                            session.status === "active" ? "bg-success-9" : "bg-warning-9",
                          )}
                        />
                        <span className="truncate">{STATUS_LABEL[session.status]}</span>
                      </span>
                      <span className="truncate">
                        <TimestampInfo value={session.createdAt} displayType="relative" />
                      </span>
                      <span className="truncate">
                        <TimestampInfo value={session.expiresAt} displayType="relative" />
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            </section>
          </SlidePanelContent>
          {canRevoke ? (
            <SlidePanelFooter className="flex items-center justify-end gap-3">
              <Button variant="outline" size="md" onClick={onClose}>
                Close
              </Button>
              <Button variant="primary" color="danger" size="md" onClick={() => onRevoke(group)}>
                Revoke all sessions
              </Button>
            </SlidePanelFooter>
          ) : null}
        </>
      ) : null}
    </SlidePanel>
  );
}

export function PortalSessionsSkeleton() {
  return (
    <ResourceList>
      <ResourceListHeader>
        <Skeleton className="h-8 w-full rounded-lg md:w-80" />
      </ResourceListHeader>
      <SessionsTableSkeleton />
    </ResourceList>
  );
}

function SessionsTableSkeleton() {
  return (
    <ResourceListContent aria-busy="true" aria-live="polite">
      <output className="sr-only">Loading sessions…</output>
      <div className="overflow-x-auto">
        <div className="min-w-[640px]">
          <SessionsTableHeader />
          <ResourceListBody aria-hidden="true">
            {SKELETON_KEYS.map((key) => (
              <ResourceListItem key={key} className={cn(GRID, "h-12")}>
                <Skeleton className="h-3 w-40" />
                <Skeleton className="h-3 w-6" />
                <Skeleton className="h-3 w-24" />
                <Skeleton className="h-3 w-24" />
                <span />
              </ResourceListItem>
            ))}
          </ResourceListBody>
        </div>
      </div>
    </ResourceListContent>
  );
}
