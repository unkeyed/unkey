import { db, Workspace } from "@/lib/db";
import { redirect } from "next/navigation";
import { parseAsArrayOf, parseAsInteger, parseAsString } from "nuqs/server";
import {
  AuditLogWithTargets,
  DEFAULT_BUCKET_NAME,
  queryAuditLogs,
} from "@/lib/trpc/routers/audit/fetch";
import { clerkClient, User } from "@clerk/nextjs/server";

export const getWorkspace = async (tenantId: string) => {
  try {
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
    return workspace;
  } catch (error) {
    console.error(
      `Failed to fetch workspace for tenant ID ${tenantId}: ${
        error instanceof Error ? error.message : "Unknown error"
      }`
    );
    throw error;
  }
};

export type SearchParams = {
  events?: string | string[];
  users?: string | string[];
  rootKeys?: string | string[];
  startTime?: string | string[];
  endTime?: any;
};

type ParseFilterInput = SearchParams & {
  bucket: string;
};

export type ParsedParams = {
  selectedEvents: string[];
  selectedUsers: string[];
  selectedRootKeys: string[];
  startTime: number | null;
  endTime: number | null;
  bucket: string | null;
};

export const parseFilterParams = (params: ParseFilterInput): ParsedParams => {
  const filterParser = parseAsArrayOf(parseAsString).withDefault([]);
  const timeParser = parseAsInteger;
  const bucketParser = parseAsString;

  return {
    selectedEvents: filterParser.parseServerSide(params.events),
    selectedUsers: filterParser.parseServerSide(params.users),
    selectedRootKeys: filterParser.parseServerSide(params.rootKeys),
    startTime: timeParser.parseServerSide(params.startTime),
    endTime: timeParser.parseServerSide(params.endTime),
    bucket: bucketParser.parseServerSide(params.bucket),
  };
};

export const getAuditLogsForBucket = async (
  workspace: Workspace,
  parsedParams: ParsedParams
) => {
  const {
    bucket,
    selectedEvents,
    selectedUsers,
    selectedRootKeys,
    startTime,
    endTime,
  } = parsedParams;

  try {
    const auditLogs = await queryAuditLogs(
      {
        users: [...selectedRootKeys, ...selectedUsers],
        bucket: bucket ?? DEFAULT_BUCKET_NAME,
        startTime,
        endTime,
        events: selectedEvents,
      },
      workspace
    );
    return auditLogs;
  } catch (error) {
    console.error(
      `Failed to fetch audit logs for bucket "${bucket}" in workspace "${workspace.id}": ` +
        `Time range: ${startTime} to ${endTime}, ` +
        `Events: ${selectedEvents}, Users: ${selectedUsers}, Root keys: ${selectedRootKeys}. ` +
        `Error: ${error instanceof Error ? error.message : "Unknown error"}`
    );
    throw error;
  }
};

export const fetchUsersFromLogs = async (
  logs: AuditLogWithTargets[]
): Promise<Record<string, User>> => {
  try {
    // Get unique user IDs from logs
    const userIds = [
      ...new Set(
        logs.filter((l) => l.actorType === "user").map((l) => l.actorId)
      ),
    ];

    // Fetch all users in parallel
    const users = await Promise.all(
      userIds.map((userId) =>
        clerkClient.users.getUser(userId).catch(() => null)
      )
    );

    // Convert array to record object
    return users.reduce((acc, user) => {
      if (user) {
        acc[user.id] = user;
      }
      return acc;
    }, {} as Record<string, User>);
  } catch (error) {
    console.error("Error fetching users:", error);
    return {};
  }
};
