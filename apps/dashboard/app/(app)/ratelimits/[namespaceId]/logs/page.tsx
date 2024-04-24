import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { notFound } from "next/navigation";

import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Loading } from "@/components/dashboard/loading";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { getRatelimitEvents } from "@/lib/tinybird";
import { Box, Check, X } from "lucide-react";
import Link from "next/link";
import { parseAsArrayOf, parseAsBoolean, parseAsIsoDateTime, parseAsString } from "nuqs";
import { Suspense } from "react";
import { Filters } from "./filter";
import { Menu } from "./menu";
export const dynamic = "force-dynamic";
export const runtime = "edge";

type Props = {
  params: {
    namespaceId: string;
  };
  searchParams: {
    after?: string;
    before?: string;
    identifier?: string | string[];
    ipAddress?: string | string[];
    country?: string | string[];
    success?: string;
  };
};

/**
 * Parse searchParam string arrays
 */
const stringParser = parseAsArrayOf(parseAsString).withDefault([]);

export default async function AuditPage(props: Props) {
  const tenantId = getTenantId();

  const namespace = await db.query.ratelimitNamespaces.findFirst({
    where: (table, { eq, and, isNull }) =>
      and(eq(table.id, props.params.namespaceId), isNull(table.deletedAt)),
    with: {
      workspace: true,
    },
  });
  if (!namespace || namespace.workspace.tenantId !== tenantId) {
    return notFound();
  }

  const selected = {
    identifier: stringParser.parseServerSide(props.searchParams.identifier),
    ipAddress: stringParser.parseServerSide(props.searchParams.ipAddress),
    country: stringParser.parseServerSide(props.searchParams.country),
    success: parseAsBoolean.parseServerSide(props.searchParams.success),
    after: parseAsIsoDateTime.parseServerSide(props.searchParams.after),
    before: parseAsIsoDateTime.parseServerSide(props.searchParams.before),
  };

  return (
    <div>
      <main className="flex flex-col gap-2 mt-8 mb-20">
        <Filters />

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
            workspaceId={namespace.workspace.id}
            namespaceId={namespace.id}
            selected={selected}
          />
        </Suspense>
      </main>
    </div>
  );
}

const AuditLogTable: React.FC<{
  workspaceId: string;
  namespaceId: string;
  selected: {
    identifier: string[];
    ipAddress: string[];
    country: string[];
    success: boolean | null;
    before: Date | null;
    after: Date | null;
  };
}> = async ({ workspaceId, namespaceId, selected }) => {
  const isFiltered =
    selected.identifier.length > 0 ||
    selected.ipAddress.length > 0 ||
    selected.country.length > 0 ||
    selected.before ||
    selected.after ||
    typeof selected.success === "boolean";

  const query = {
    workspaceId: workspaceId,
    namespaceId: namespaceId,
    before: selected.before?.getTime() ?? undefined,
    after: selected.after?.getTime() ?? undefined,
    identifier: selected.identifier.length > 0 ? selected.identifier : undefined,
    country: selected.country.length > 0 ? selected.country : undefined,
    ipAddress: selected.ipAddress.length > 0 ? selected.ipAddress : undefined,

    success: selected.success ?? undefined,
  };
  const logs = await getRatelimitEvents(query).catch((err) => {
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
            <Link href="/audit" prefetch>
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

  return (
    <div>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Time</TableHead>
            <TableHead>Identifier</TableHead>
            <TableHead>Success</TableHead>
            <TableHead>Remaining</TableHead>
            <TableHead>IP address</TableHead>
            <TableHead>Country</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {logs.data.map((l) => (
            <TableRow key={l.requestId}>
              <TableCell>
                <span className="text-sm text-content">{new Date(l.time).toISOString()}</span>
              </TableCell>

              <TableCell>
                <Badge variant="secondary" className="font-mono text-xs">
                  {l.identifier}
                </Badge>
              </TableCell>
              <TableCell>
                <span className="font-mono text-xs text-content">
                  {l.success ? (
                    <Check className="w-4 h-4" />
                  ) : (
                    <X className="w-4 h-4 text-content-alert" />
                  )}
                </span>
              </TableCell>
              <TableCell>
                <Badge variant="secondary">
                  {l.remaining} / {l.limit}
                </Badge>
              </TableCell>
              <TableCell>
                <pre className="text-xs text-content-subtle">{l.ipAddress} </pre>
              </TableCell>
              <TableCell>
                <pre className="text-xs text-content-subtle">{l.country} </pre>
              </TableCell>
              <TableCell>
                <Menu namespace={{ id: namespaceId }} identifier={l.identifier} />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
};
