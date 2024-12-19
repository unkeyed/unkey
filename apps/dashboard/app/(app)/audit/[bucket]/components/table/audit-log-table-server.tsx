"use server";

import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { AuditLogWithTargets } from "@/lib/trpc/routers/audit/fetch";
import { Box } from "lucide-react";
import { fetchUsersFromLogs } from "../../actions";
import Link from "next/link";
import { Button } from "@unkey/ui";
import { AuditLogTableClient } from "./audit-log-table-client";

export const AuditLogTableServer: React.FC<{
  selectedEvents: string[];
  selectedUsers: string[];
  selectedRootKeys: string[];
  logs: AuditLogWithTargets[];
}> = async ({ selectedEvents, selectedRootKeys, selectedUsers, logs }) => {
  const isFiltered =
    selectedEvents.length > 0 ||
    selectedUsers.length > 0 ||
    selectedRootKeys.length > 0;

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

  const users = await fetchUsersFromLogs(logs);

  // INFO: Without that json.parse and stringify next.js goes brrrrr
  return (
    <AuditLogTableClient
      data={logs}
      users={JSON.parse(JSON.stringify(users))}
    />
  );
};
