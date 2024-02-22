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

import { Button } from "@/components/ui/button";
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
import { ExternalLink, KeySquare, Minus } from "lucide-react";
import Link from "next/link";
import { Filter } from "./filter";
export const dynamic = "force-dynamic";
export const runtime = "edge";

type Props = {
  searchParams: {
    before?: number;
    events?: string;
  };
};

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
  const selectedEvents = props.searchParams.events
    ? props.searchParams.events.split(",")
    : undefined;

  const retentionCutoff =
    Date.now() - workspace.betaFeatures.auditLogRetentionDays * 24 * 60 * 60 * 1000;

  console.log({ selectedEvents });

  const logs = await getAuditLogs({
    workspaceId: workspace.id,
    before: props.searchParams.before ? Number(props.searchParams.before) : undefined,
    after: retentionCutoff,
    events: selectedEvents,
  }).catch((err) => {
    console.error(err);
    throw err;
  });

  const hasPreviousPage = !!props.searchParams.before;
  const hasNextPage = logs.rows_before_limit_at_least;

  function buildHref(override: Partial<Props["searchParams"]>): string {
    const searchParams = new URLSearchParams();
    const before = override.before ?? props.searchParams.before;
    if (before) {
      searchParams.set("before", before.toString());
    }

    const events = override.events ?? props.searchParams.events;
    if (events) {
      searchParams.set("events", events);
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
        <Filter
          options={[
            {
              value: "key.create",
              label: "key.create",
            },
          ]}
          selected={selectedEvents ?? []}
        />
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Time</TableHead>
              <TableHead>Event</TableHead>
              <TableHead>Actor</TableHead>
              <TableHead>IP address</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {logs.data.map((l) => {
              const user = users[l.actorId];
              return (
                <TableRow key={l.auditLogId}>
                  <TableCell>
                    <div className="flex flex-col">
                      <span className="text-content">{new Date(l.time).toDateString()}</span>
                      <span className="text-xs text-content-subtle">
                        {new Date(l.time).toTimeString()}
                      </span>
                    </div>
                  </TableCell>

                  <TableCell>
                    <Badge variant="secondary">{l.event}</Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center">
                      {user ? (
                        <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs md:flex-grow">
                          <Avatar className="w-6 h-6">
                            <AvatarImage src={user.imageUrl} />
                            <AvatarFallback>{user.username?.slice(0, 2)}</AvatarFallback>
                          </Avatar>
                          <span className="text-content">{`${user.firstName} ${user.lastName}`}</span>
                        </div>
                      ) : (
                        <div className="flex items-center w-full gap-2 max-sm:m-0 max-sm:gap-1 max-sm:text-xs md:flex-grow">
                          <KeySquare className="w-4 h-4" />
                          <span className="font-mono text-xs text-content">{l.actorId}</span>
                        </div>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    {l.ipAddress ? (
                      <pre className="text-xs text-content-subtle">{l.ipAddress}</pre>
                    ) : (
                      <Minus className="w-4 h-4 text-content-subtle" />
                    )}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>

        <div className="w-full mt-8">
          <Button
            size="block"
            disabled={logs.data.length < 100}
            variant={logs.data.length < 100 ? "disabled" : "secondary"}
          >
            <Link href={buildHref({ before: logs.data?.at(-1)?.time })}>Load more</Link>
          </Button>
        </div>
      </main>
    </div>
  );
}
