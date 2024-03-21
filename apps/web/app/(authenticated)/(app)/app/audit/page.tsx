import { PageHeader } from "@/components/dashboard/page-header";
import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { clerkClient } from "@clerk/nextjs";
import { User } from "@clerk/nextjs/server";
import { notFound } from "next/navigation";

import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { getAuditLogs } from "@/lib/tinybird";
import { unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import { Box, X } from "lucide-react";
import Link from "next/link";
import { parseAsArrayOf, parseAsString } from "nuqs";
import { Suspense } from "react";
import { Filter } from "./filter";
import { Row } from "./row";
import { FilterSingle } from "./select";
export const dynamic = "force-dynamic";
export const runtime = "edge";

type Props = {
  searchParams: {
    before?: number;
    events?: string | string[];
    users?: string | string[];
    rootKeys?: string | string[];
    bucket?: string;
  };
};

/**
 * Parse searchParam string arrays
 */
const filterParser = parseAsArrayOf(parseAsString).withDefault([]);

export default async function AuditPage(props: Props) {
  const tenantId = getTenantId();
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
    with: {
      ratelimitNamespaces: {
        columns: {
          id: true,
          name: true,
        },
      },
    },
  });
  if (!workspace?.betaFeatures.auditLogRetentionDays) {
    return notFound();
  }
  const selectedBucket = parseAsString
    .withDefault("unkey_mutations")
    .parseServerSide(props.searchParams.bucket);
  const selectedEvents = filterParser.parseServerSide(props.searchParams.events);
  const selectedUsers = filterParser.parseServerSide(props.searchParams.users);
  const selectedRootKeys = filterParser.parseServerSide(props.searchParams.rootKeys);

  return (
    <div>
      <PageHeader
        title="Audit Logs"
        description={`You have access to the last ${workspace.betaFeatures.auditLogRetentionDays} days.`}
      />

      <main className="mt-8 mb-20">
        <div className="flex items-center justify-start gap-2 my-4">
          <FilterSingle
            param="bucket"
            title="Bucket"
            options={[
              { value: "unkey_mutations", label: "System" },
              ...workspace.ratelimitNamespaces.map((ns) => ({
                value: `ratelimit.${ns.id}`,
                label: ns.name,
              })),
            ]}
          />
          {selectedBucket === "unkey_mutations" ? (
            <Filter
              param="events"
              title="Events"
              options={Object.values(unkeyAuditLogEvents.Values).map((value) => ({
                value,
                label: value,
              }))}
            />
          ) : (
            <FilterSingle
              param="event"
              title="Event"
              options={[
                { value: "ratelimit.success", label: "Ratelimit success" },
                { value: "ratelimit.denied", label: "Ratelimit denied" },
              ]}
            />
          )}
          {selectedBucket === "unkey_mutations" ? (
            <Suspense fallback={<Filter param="users" title="Users" options={[]} />}>
              <UserFilter tenantId={workspace.tenantId} />
            </Suspense>
          ) : null}
          {selectedBucket === "unkey_mutations" ? (
            <Suspense fallback={<Filter param="rootKeys" title="Root Keys" options={[]} />}>
              <RootKeyFilter workspaceId={workspace.id} />
            </Suspense>
          ) : null}
          {selectedEvents.length > 0 || selectedUsers.length > 0 || selectedRootKeys.length > 0 ? (
            <Link href="/app/audit">
              <Button
                variant="outline"
                size="sm"
                className="flex items-center h-8 gap-2 bg-background-subtle"
              >
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
          <AuditLogTable
            workspaceId={workspace.id}
            before={props.searchParams.before ? Number(props.searchParams.before) : undefined}
            after={Date.now() - workspace.betaFeatures.auditLogRetentionDays * 24 * 60 * 60 * 1000}
            selectedEvents={selectedEvents}
            selectedUsers={selectedUsers}
            selectedRootKeys={selectedRootKeys}
            selectedBucket={selectedBucket}
          />
        </Suspense>
      </main>
    </div>
  );
}

const AuditLogTable: React.FC<{
  workspaceId: string;
  selectedEvents: string[];
  selectedUsers: string[];
  selectedRootKeys: string[];
  selectedBucket: string;
  before?: number;
  after: number;
}> = async ({
  workspaceId,
  selectedEvents,
  selectedRootKeys,
  selectedUsers,
  selectedBucket,
  before,
  after,
}) => {
  const isFiltered =
    selectedEvents.length > 0 || selectedUsers.length > 0 || selectedRootKeys.length > 0 || before;
  const logs = await getAuditLogs({
    workspaceId: workspaceId,
    before,
    after,
    bucket: selectedBucket,
    events: selectedEvents.length > 0 ? selectedEvents : undefined,
    actorIds:
      selectedUsers.length > 0 || selectedRootKeys.length > 0
        ? [...selectedUsers, ...selectedRootKeys]
        : undefined,
  }).catch((err) => {
    console.error(err);
    throw err;
  });

  if (logs.data.length === 0) {
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
    );
  }

  const hasMoreLogs = logs.data.length >= 100;

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
    return `/app/audit?${searchParams.toString()}`;
  }

  const userIds = [
    ...new Set(logs.data.filter((l) => l.actor.type === "user").map((l) => l.actor.id)),
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
          {logs.data.map((l) => {
            const user = users[l.actor.id];
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
                  actor: l.actor,
                  event: l.event,
                  location: l.context.location,
                  resources: l.resources,
                  description: l.description,
                }}
              />
            );
          })}
        </TableBody>
      </Table>

      <div className="w-full mt-8">
        <Link href={buildHref({ before: logs.data?.at(-1)?.time })} prefetch>
          <Button
            size="block"
            disabled={!hasMoreLogs}
            variant={hasMoreLogs ? "secondary" : "disabled"}
          >
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
    where: (table, { eq }) => eq(table.forWorkspaceId, workspaceId),
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
