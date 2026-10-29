"use client";

import { ListPager } from "@/components/list-pager";
import { LoadError } from "@/components/load-error";
import type { User } from "@/lib/auth/types";
import { type CliSession, listCliSessions, revokeCliSession } from "@/lib/cli/device-api";
import { listOf } from "@/lib/loadable/list";
import { fromQuery } from "@/lib/loadable/loadable";
import { getErrorMessage } from "@/lib/unkey-client";
import { useWorkspace } from "@/providers/workspace-provider";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { IconLaptop2Outline18, IconLink4Outline18, IconTerminalOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Badge,
  Button,
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  InfoTooltip,
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemHeader,
  ItemMedia,
  ItemSeparator,
  ItemTitle,
  PageBody,
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  Skeleton,
  TimestampInfo,
  toast,
} from "@unkey/ui";
import { cn } from "cn";
import { Fragment, useState } from "react";

type SessionState = "active" | "expired" | "revoked";

const STATES: Record<SessionState, { badge: string | null; label: string }> = {
  active: { badge: null, label: "Expires" },
  expired: { badge: "Expired", label: "Expired" },
  revoked: { badge: "Revoked", label: "Revoked" },
};

function sessionState(session: CliSession): SessionState {
  if (session.revokedAt) {
    return "revoked";
  }
  return session.expiresAt <= Date.now() ? "expired" : "active";
}

type SessionHolder = { kind: "cli" | "app"; name: string };

function sessionHolder(session: CliSession): SessionHolder {
  if (session.clientId) {
    return { kind: "app", name: session.deviceName || "Unknown app" };
  }
  return { kind: "cli", name: session.deviceName || "Unknown device" };
}

export type CliSessionOwner = Pick<User, "id" | "fullName" | "email">;

export function CliSessions({ owners }: { owners: CliSessionOwner[] }) {
  const { user } = useWorkspace();
  const ownersById = new Map(owners.map((owner) => [owner.id, owner]));

  return (
    <PageContainer>
      <PageHeader className="max-w-[920px]">
        <PageHeaderContent>
          <PageHeaderTitle>Sessions</PageHeaderTitle>
        </PageHeaderContent>
      </PageHeader>
      <PageBody className="max-w-[920px]">
        <CliSessionResults
          orgId={user?.orgId}
          ownersById={ownersById}
          isAdmin={user?.role === "admin"}
        />
      </PageBody>
    </PageContainer>
  );
}

function CliSessionResults({
  orgId,
  ownersById,
  isAdmin,
}: {
  orgId?: string | null;
  ownersById: ReadonlyMap<string, CliSessionOwner>;
  isAdmin: boolean;
}) {
  const queryClient = useQueryClient();
  const queryKey = ["cli-sessions", orgId];
  const view = listOf(fromQuery(useQuery({ queryKey, queryFn: listCliSessions })));
  const [confirming, setConfirming] = useState<CliSession | null>(null);
  const revoke = useMutation({
    mutationFn: revokeCliSession,
    async onSuccess() {
      toast.success("Session revoked.");
      await queryClient.invalidateQueries({ queryKey });
    },
    onError(error) {
      toast.error(getErrorMessage(error, "Could not revoke this session."));
    },
  });

  const renderGroup = (title: string, dot: string, sessions: readonly CliSession[]) =>
    sessions.length === 0 ? null : (
      <ItemGroup variant="outline" aria-label={title}>
        <ItemHeader>
          <ItemContent>
            <div className="flex items-center gap-2">
              <span aria-hidden="true" className={cn("size-1.5 rounded-full", dot)} />
              <ItemTitle>{title}</ItemTitle>
            </div>
          </ItemContent>
        </ItemHeader>
        {sessions.map((session) => (
          <Fragment key={session.id}>
            <ItemSeparator />
            <SessionItem
              session={session}
              owner={isAdmin ? (ownersById.get(session.userId) ?? "unknown") : "hidden"}
              disabled={revoke.isLoading}
              onRevoke={() => setConfirming(session)}
            />
          </Fragment>
        ))}
      </ItemGroup>
    );

  return (
    <>
      {match(view)
        .with({ status: "loading" }, () => <CliSessionsSkeleton />)
        .with({ status: "failed" }, ({ error, retry }) => (
          <ItemGroup variant="outline">
            <LoadError title="Could not load sessions" error={error} frame="none" onRetry={retry} />
          </ItemGroup>
        ))
        .with({ status: "empty" }, () => <CliSessionsEmpty />)
        .with({ status: "ready" }, ({ rows, footer }) => (
          <>
            {renderGroup(
              "Active",
              "bg-success-9",
              rows.filter((session) => sessionState(session) === "active"),
            )}
            {renderGroup(
              "Expired and revoked",
              "bg-gray-8",
              rows.filter((session) => sessionState(session) !== "active"),
            )}
            <ListPager state={footer} noun="sessions" />
          </>
        ))
        .exhaustive()}
      <RevokeSessionDialog
        session={confirming}
        revoking={revoke.isLoading}
        onCancel={() => setConfirming(null)}
        onConfirm={(session) => revoke.mutate(session.id, { onSuccess: () => setConfirming(null) })}
      />
    </>
  );
}

function SessionOwner({ owner }: { owner: CliSessionOwner }) {
  if (!owner.fullName) {
    return <span className="secret">{owner.email}</span>;
  }
  return (
    <InfoTooltip content={owner.email} position={{ side: "top" }}>
      <span className="secret cursor-default text-gray-12">{owner.fullName}</span>
    </InfoTooltip>
  );
}

function SessionItem({
  session,
  owner,
  disabled,
  onRevoke,
}: {
  session: CliSession;
  owner: CliSessionOwner | "unknown" | "hidden";
  disabled: boolean;
  onRevoke: () => void;
}) {
  const state = sessionState(session);
  const { badge, label } = STATES[state];
  const endsAt = session.revokedAt ?? session.expiresAt;
  const holder = sessionHolder(session);

  return (
    <Item>
      <ItemMedia className="bg-grayA-3 text-gray-11">
        {holder.kind === "app" ? <IconLink4Outline18 /> : <IconLaptop2Outline18 />}
      </ItemMedia>
      <ItemContent>
        <ItemTitle className="flex items-center gap-2">
          <span className="truncate">{holder.name}</span>
          <Badge variant="secondary" size="sm">
            {holder.kind === "app" ? "App" : "CLI"}
          </Badge>
        </ItemTitle>
        <ItemDescription>
          {match(owner)
            .with("hidden", () => null)
            .with("unknown", () => (
              <>
                <span>Unknown user</span>
                {" · "}
              </>
            ))
            .otherwise((known) => (
              <>
                <SessionOwner owner={known} />
                {" · "}
              </>
            ))}
          Last used <TimestampInfo value={session.lastUsedAt} displayType="relative" />
          {" · "}
          {label} <TimestampInfo value={endsAt} displayType="relative" />
          {" · "}
          {session.permissions.length} permission{session.permissions.length === 1 ? "" : "s"}
        </ItemDescription>
      </ItemContent>
      <ItemActions>
        {badge ? (
          <Badge variant="secondary" size="sm">
            {badge}
          </Badge>
        ) : (
          <Button
            type="button"
            variant="outline"
            aria-label={`Revoke ${holder.name}`}
            disabled={disabled}
            onClick={onRevoke}
          >
            Revoke
          </Button>
        )}
      </ItemActions>
    </Item>
  );
}

function RevokeSessionDialog({
  session,
  revoking,
  onCancel,
  onConfirm,
}: {
  session: CliSession | null;
  revoking: boolean;
  onCancel: () => void;
  onConfirm: (session: CliSession) => void;
}) {
  const holder = session ? sessionHolder(session) : null;
  return (
    <AlertDialog
      open={session !== null}
      onOpenChange={(open) => {
        if (!open && !revoking) {
          onCancel();
        }
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {holder?.kind === "app" ? "Revoke this app session?" : "Revoke this CLI session?"}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {holder?.kind === "app" ? (
              <>
                <span className="font-medium text-gray-12">{holder.name}</span> loses access right
                away.
              </>
            ) : (
              <>
                The CLI on{" "}
                <span className="font-medium text-gray-12">
                  {session?.deviceName || "this unknown device"}
                </span>{" "}
                loses access right away.
              </>
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={revoking}>Cancel</AlertDialogCancel>
          <Button
            type="button"
            variant="primary"
            color="danger"
            loading={revoking}
            loadingLabel="Revoking session"
            onClick={() => {
              if (session) {
                onConfirm(session);
              }
            }}
          >
            Revoke session
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

const SKELETON_ROWS = ["first", "second", "third"] as const;

function CliSessionsSkeleton() {
  return (
    <ItemGroup variant="outline" aria-busy="true">
      <output className="sr-only">Loading sessions...</output>
      <div aria-hidden="true">
        {SKELETON_ROWS.map((row, index) => (
          <Fragment key={row}>
            {index > 0 ? <ItemSeparator /> : null}
            <Item>
              <Skeleton className="size-8 shrink-0 rounded-md" />
              <ItemContent className="gap-2">
                <Skeleton className="h-3 w-40" />
                <Skeleton className="h-2.5 w-64" />
              </ItemContent>
              <Skeleton className="h-8 w-18 rounded-md" />
            </Item>
          </Fragment>
        ))}
      </div>
    </ItemGroup>
  );
}

function CliSessionsEmpty() {
  return (
    <ItemGroup variant="outline">
      <EmptyState frame="none">
        <EmptyStateIcon>
          <IconTerminalOutline18 />
        </EmptyStateIcon>
        <EmptyStateHeader>
          <EmptyStateTitle>No sessions yet</EmptyStateTitle>
          <EmptyStateDescription>
            Run{" "}
            <code className="rounded-md bg-grayA-3 px-1.5 py-0.5 font-mono text-xs text-gray-12">
              unkey login
            </code>{" "}
            to start one.
          </EmptyStateDescription>
        </EmptyStateHeader>
      </EmptyState>
    </ItemGroup>
  );
}
