import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Loading } from "@/components/dashboard/loading";
import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { Table, TableBody, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { clerkClient } from "@clerk/nextjs";
import type { User } from "@clerk/nextjs/server";
import type { SelectAuditLog, SelectAuditLogTarget } from "@unkey/db/src/schema";
import { InputSearch } from "@unkey/icons";
import { unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import { Button } from "@unkey/ui";
import { Box, X } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import { parseAsArrayOf, parseAsString } from "nuqs/server";
import { Suspense } from "react";
import { BucketSelect } from "./bucket-select";
import { Filter } from "./filter";
import { Row } from "./row";

export const dynamic = "force-dynamic";
export const runtime = "edge";

type Props = {
  params: {
    bucket: string;
  };
  searchParams: {
    before?: number;
    events?: string | string[];
    users?: string | string[];
    rootKeys?: string | string[];
  };
};

type AuditLogWithTargets = SelectAuditLog & {
  targets: Array<SelectAuditLogTarget>;
};

/**
 * Parse searchParam string arrays
 */
const filterParser = parseAsArrayOf(parseAsString).withDefault([]);

/**
 * Utility to map log with targets to log entry
 */
const toLogEntry = (l: AuditLogWithTargets) => ({
  id: l.id,
  event: l.event,
  time: l.time,
  actor: {
    id: l.actorId,
    name: l.actorName,
    type: l.actorType,
  },
  location: l.remoteIp,
  description: l.display,
  targets: l.targets.map((t) => ({
    id: t.id,
    type: t.type,
    name: t.name,
  })),
});

export default async function AuditPage(props: Props) {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      ratelimitNamespaces: {
        where: (table, { isNull }) => isNull(table.deletedAt),
        columns: {
          id: true,
          name: true,
        },
      },
    },
  });
  if (!workspace) {
    return redirect("/auth/signin");
  }

  const selectedEvents = filterParser.parseServerSide(props.searchParams.events);
  const selectedUsers = filterParser.parseServerSide(props.searchParams.users);
  const selectedRootKeys = filterParser.parseServerSide(props.searchParams.rootKeys);

  /**
   * If not specified, default to 30 days
   */
  const retentionDays =
    workspace.features.auditLogRetentionDays ?? workspace.plan === "free" ? 30 : 90;
  const retentionCutoffUnixMilli = Date.now() - retentionDays * 24 * 60 * 60 * 1000;

  const selectedActorIds = [...selectedRootKeys, ...selectedUsers];

  const bucket = await db.query.auditLogBucket.findFirst({
    where: (table, { eq, and }) =>
      and(eq(table.workspaceId, workspace.id), eq(table.name, props.params.bucket)),
    with: {
      logs: {
        where: (table, { and, inArray, gte }) =>
          and(
            selectedEvents.length > 0 ? inArray(table.event, selectedEvents) : undefined,
            gte(table.createdAt, retentionCutoffUnixMilli),
            selectedActorIds.length > 0 ? inArray(table.actorId, selectedActorIds) : undefined,
          ),

        with: {
          targets: true,
        },
        orderBy: (table, { desc }) => desc(table.time),
        limit: 100,
      },
    },
  });

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<InputSearch />}>
          <Navbar.Breadcrumbs.Link href="/audit/unkey_mutations">Audit</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={`/audit/${props.params.bucket}`} active isIdentifier>
            {workspace.ratelimitNamespaces.find((ratelimit) => ratelimit.id === props.params.bucket)
              ?.name ?? props.params.bucket}
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
      </Navbar>
      <PageContent>
        <main className="mb-20">
          <div className="flex items-center justify-start gap-2 mb-4">
            <BucketSelect
              selected={props.params.bucket}
              ratelimitNamespaces={workspace.ratelimitNamespaces}
            />
            <Filter
              param="events"
              title="Events"
              options={
                props.params.bucket === "unkey_mutations"
                  ? Object.values(unkeyAuditLogEvents.Values).map((value) => ({
                      value,
                      label: value,
                    }))
                  : [
                      {
                        value: "ratelimit.success",
                        label: "Ratelimit success",
                      },
                      { value: "ratelimit.denied", label: "Ratelimit denied" },
                    ]
              }
            />

            {props.params.bucket === "unkey_mutations" ? (
              <Suspense fallback={<Filter param="users" title="Users" options={[]} />}>
                <UserFilter tenantId={workspace.tenantId} />
              </Suspense>
            ) : null}
            <Suspense fallback={<Filter param="rootKeys" title="Root Keys" options={[]} />}>
              <RootKeyFilter workspaceId={workspace.id} />
            </Suspense>
            {selectedEvents.length > 0 ||
            selectedUsers.length > 0 ||
            selectedRootKeys.length > 0 ? (
              <Link href="/audit">
                <Button className="flex items-center h-8 gap-2 bg-background-subtle">
                  Clear
                  <X className="w-4 h-4" />
                </Button>
              </Link>
            ) : null}
          </div>
          <Suspense
            fallback={
              <EmptyPlaceholder>
                <EmptyPlaceholder.Icon>
                  <Loading />
                </EmptyPlaceholder.Icon>
              </EmptyPlaceholder>
            }
          >
            {!bucket ? (
              <EmptyPlaceholder>
                <EmptyPlaceholder.Icon>
                  <Box />
                </EmptyPlaceholder.Icon>
                <EmptyPlaceholder.Title>Bucket Not Found</EmptyPlaceholder.Title>
                <EmptyPlaceholder.Description>
                  The specified audit log bucket does not exist or you do not have access to it.
                </EmptyPlaceholder.Description>
              </EmptyPlaceholder>
            ) : (
              <AuditLogTable
                logs={bucket.logs.map(toLogEntry)}
                before={props.searchParams.before ? Number(props.searchParams.before) : undefined}
                selectedEvents={selectedEvents}
                selectedUsers={selectedUsers}
                selectedRootKeys={selectedRootKeys}
              />
            )}
          </Suspense>
        </main>
      </PageContent>
    </div>
  );
}

const AuditLogTable: React.FC<{
  selectedEvents: string[];
  selectedUsers: string[];
  selectedRootKeys: string[];
  before?: number;
  logs: Array<{
    id: string;
    event: string;
    time: number;
    actor: {
      id: string;
      type: string;
      name: string | null;
    };
    location: string | null;
    description: string;
    targets: Array<{
      id: string;
      type: string;
      name: string | null;
    }>;
  }>;
}> = async ({ selectedEvents, selectedRootKeys, selectedUsers, before, logs }) => {
  const isFiltered =
    selectedEvents.length > 0 || selectedUsers.length > 0 || selectedRootKeys.length > 0 || before;

  if (logs.length === 0) {
    return (
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
            <Link href="/audit" prefetch>
              <Button>Reset Filters</Button>
            </Link>
          </div>
        ) : (
          <EmptyPlaceholder.Description>
            Create, update or delete something and come back again.
          </EmptyPlaceholder.Description>
        )}
      </EmptyPlaceholder>
    );
  }

  const hasMoreLogs = logs.length >= 100;

  function buildHref(override: Partial<Props["searchParams"]>): string {
    const searchParams = new URLSearchParams();
    const newBefore = override.before ?? before;
    if (newBefore) {
      searchParams.set("before", newBefore.toString());
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
    return `/audit?${searchParams.toString()}`;
  }

  const userIds = [...new Set(logs.filter((l) => l.actor.type === "user").map((l) => l.actor.id))];
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
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Actor</TableHead>
            <TableHead>Event</TableHead>
            <TableHead>Location</TableHead>
            <TableHead>Time</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {logs.map((l) => {
            const user = users[l.actor.id];
            return (
              <Row
                key={l.id}
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
                  actor: l.actor,
                  event: l.event,
                  location: l.location,
                  targets: l.targets,
                  description: l.description,
                }}
              />
            );
          })}
        </TableBody>
      </Table>

      <div className="w-full mt-8">
        <Link href={buildHref({ before: logs.at(-1)?.time })} prefetch>
          <Button className="block" disabled={!hasMoreLogs}>
            {hasMoreLogs ? "Load more" : "No more logs"}
          </Button>
        </Link>
      </div>
    </div>
  );
};

const UserFilter: React.FC<{ tenantId: string }> = async ({ tenantId }) => {
  if (tenantId.startsWith("user_")) {
    return null;
  }
  const members = await clerkClient.organizations.getOrganizationMembershipList({
    organizationId: tenantId,
  });

  return (
    <Filter
      param="users"
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
    />
  );
};

const RootKeyFilter: React.FC<{ workspaceId: string }> = async ({ workspaceId }) => {
  const rootKeys = await db.query.keys.findMany({
    where: (table, { eq, and, or, isNull, gt }) =>
      and(
        eq(table.forWorkspaceId, workspaceId),
        isNull(table.deletedAt),
        or(isNull(table.expires), gt(table.expires, new Date())),
      ),
    columns: {
      id: true,
      name: true,
    },
  });

  return (
    <Filter
      param="rootKeys"
      title="Root Keys"
      options={rootKeys.map((k) => ({
        label: k.id ?? k.id,
        value: k.id,
      }))}
    />
  );
};
