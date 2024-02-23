import { PageHeader } from "@/components/dashboard/page-header";
import { getTenantId } from "@/lib/auth";
import { db, desc, schema } from "@/lib/db";
import { clerkClient } from "@clerk/nextjs";
import { User } from "@clerk/nextjs/server";
import { notFound } from "next/navigation";

import { CopyButton } from "@/components/dashboard/copy-button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";

import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Button } from "@/components/ui/button";
import { Code } from "@/components/ui/code";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { getAuditLogs } from "@/lib/tinybird";
import { Tooltip } from "@radix-ui/react-tooltip";
import { Box, ExternalLink, KeySquare, Minus } from "lucide-react";
import Link from "next/link";
import { Suspense } from "react";
import { Filter } from "./filter";
import { Row } from "./row";
export const dynamic = "force-dynamic";
export const runtime = "edge";

type Props = {
  searchParams: {
    before?: number;
    event?: string | string[];
    user?: string | string[];
    rootKey?: string | string[];
  };
};

/**
 * params are weird, they can be whatever, but we want string arrays
 */
function parseParamStringToArray(s?: string | string[]): string[] {
  if (Array.isArray(s)) {
    return s;
  }
  if (!s || s.length === 0) {
    return [];
  }
  return [s];
}

export default async function AuditPage(props: Props) {
  console.log(JSON.stringify(props));
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  if (!workspace?.betaFeatures.auditLogRetentionDays) {
    return notFound();
  }
  const retentionCutoff =
    Date.now() - workspace.betaFeatures.auditLogRetentionDays * 24 * 60 * 60 * 1000;

  const selectedEvents = parseParamStringToArray(props.searchParams.event);
  const selectedUsers = parseParamStringToArray(props.searchParams.user);
  const selectedRootKeys = parseParamStringToArray(props.searchParams.rootKey);

  const isFiltered =
    selectedEvents.length > 0 ||
    selectedUsers.length > 0 ||
    selectedRootKeys.length > 0 ||
    props.searchParams.before;

  console.log({ selectedEvents, selectedRootKeys, selectedUsers });
  const logs = await getAuditLogs({
    workspaceId: workspace.id,
    before: props.searchParams.before ? Number(props.searchParams.before) : undefined,
    after: retentionCutoff,
    events: selectedEvents.length > 0 ? selectedEvents : undefined,
  }).catch((err) => {
    console.error(err);
    throw err;
  });

  const hasMoreLogs = logs.data.length >= 100;

  function buildHref(override: Partial<Props["searchParams"]>): string {
    const searchParams = new URLSearchParams();
    const before = override.before ?? props.searchParams.before;
    if (before) {
      searchParams.set("before", before.toString());
    }

    for (const event of selectedEvents) {
      searchParams.append("event", event);
    }

    for (const rootKey of selectedRootKeys) {
      searchParams.append("rootKey", rootKey);
    }

    for (const user of selectedUsers) {
      searchParams.append("user", user);
    }

    return `/app/audit?${searchParams.toString()}`;
  }

  const userIds = [
    ...new Set(logs.data.filter((l) => l.actorType === "user").map((l) => l.actorId)),
  ];
  const users = (
    await Promise.all(userIds.map((userId) => clerkClient.users.getUser(userId).catch(() => null)))
  ).reduce(
    (acc, u) => {
      if (u) {
        acc[u.id] = u;
      }
      return acc;
    },
    {} as Record<string, User>,
  );
  return (
    <div>
      <PageHeader
        title="Audit Logs"
        description={`You have access to the last ${workspace.betaFeatures.auditLogRetentionDays} days.`}
      />

      <main className="mt-8 mb-20">
        <div className="flex items-center justify-start gap-2 my-4">
          <Filter
            param="event"
            title="Events"
            options={[
              "workspace.create",
              "workspace.update",
              "workspace.delete",
              "api.create",
              "api.update",
              "api.delete",
              "key.create",
              "key.update",
              "key.delete",
              "vercelIntegration.create",
              "vercelIntegration.update",
              "vercelIntegration.delete",
              "vercelBinding.create",
              "vercelBinding.update",
              "vercelBinding.delete",
              "role.create",
              "role.update",
              "role.delete",
              "permission.create",
              "permission.update",
              "permission.delete",
              "authorization.connect_role_and_permission",
              "authorization.disconnect_role_and_permissions",
              "authorization.connect_role_and_key",
              "authorization.disconnect_role_and_key",
              "authorization.connect_permission_and_key",
              "authorization.disconnect_permission_and_key",
            ].map((value) => ({ value, label: value }))}
            selected={selectedEvents}
          />
          <Suspense fallback={null}>
            <UserFilter tenantId={workspace.tenantId} selected={selectedUsers} />
          </Suspense>
          <Suspense fallback={null}>
            <RootKeyFilter workspaceId={workspace.id} selected={selectedRootKeys} />
          </Suspense>
        </div>
        {logs.data.length === 0 ? (
          <EmptyPlaceholder>
            <EmptyPlaceholder.Icon>
              <Box />
            </EmptyPlaceholder.Icon>
            <EmptyPlaceholder.Title>No logs found</EmptyPlaceholder.Title>
            {isFiltered ? (
              <div className="flex flex-col items-center gap-2">
                <EmptyPlaceholder.Description>
                  No events matched these filters, try changing them.{" "}
                </EmptyPlaceholder.Description>
                <Link href="/app/audit" prefetch>
                  <Button variant="secondary">Reset Filters</Button>
                </Link>
              </div>
            ) : (
              <EmptyPlaceholder.Description>
                Create, update or delete something and come back again.
              </EmptyPlaceholder.Description>
            )}
          </EmptyPlaceholder>
        ) : (
          <div>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Actor</TableHead>
                  <TableHead>Event</TableHead>
                  <TableHead>IP address</TableHead>
                  <TableHead>Time</TableHead>
                  <TableHead />
                </TableRow>
              </TableHeader>
              <TableBody>
                {logs.data.map((l) => {
                  const user = users[l.actorId];
                  return (
                    <Row
                      key={l.auditLogId}
                      user={
                        user
                          ? {
                              username: user.username,
                              firstName: user.firstName,
                              lastName: user.lastName,
                              imageUrl: user.imageUrl,
                            }
                          : undefined
                      }
                      auditLog={{
                        time: l.time,
                        actorId: l.actorId,
                        event: l.event,
                        ipAddress: l.ipAddress,
                        resources: l.resources,
                      }}
                    />
                  );
                })}
              </TableBody>
            </Table>

            <div className="w-full mt-8">
              <Button
                size="block"
                disabled={!hasMoreLogs}
                variant={hasMoreLogs ? "secondary" : "disabled"}
              >
                <Link href={buildHref({ before: logs.data?.at(-1)?.time })}>
                  {hasMoreLogs ? "Load more" : "No more logs"}
                </Link>
              </Button>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}

const UserFilter: React.FC<{ tenantId: string; selected: string[] }> = async ({
  tenantId,
  selected,
}) => {
  if (tenantId.startsWith("user_")) {
    return null;
  }
  const members = await clerkClient.organizations.getOrganizationMembershipList({
    organizationId: tenantId,
  });

  return (
    <Filter
      param="user"
      title="Users"
      options={members
        .filter((m) => Boolean(m.publicUserData))
        .map((m) => ({
          label:
            m.publicUserData!.firstName && m.publicUserData!.lastName
              ? `${m.publicUserData!.firstName} ${m.publicUserData!.lastName}`
              : m.publicUserData!.identifier,
          value: m.publicUserData!.userId,
        }))}
      selected={selected}
    />
  );
};

const RootKeyFilter: React.FC<{ workspaceId: string; selected: string[] }> = async ({
  selected,
  workspaceId,
}) => {
  const rootKeys = await db.query.keys.findMany({
    where: (table, { eq }) => eq(table.forWorkspaceId, workspaceId),
    columns: {
      id: true,
      name: true,
    },
  });

  return (
    <Filter
      param="rootKey"
      title="Root Keys"
      options={rootKeys.map((k) => ({
        label: k.id ?? k.id,
        value: k.id,
      }))}
      selected={selected}
    />
  );
};
