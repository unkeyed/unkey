import { db } from "@/lib/db";
import type { AuditLogWithTargets } from "@/lib/trpc/routers/audit/fetch";
import { type User, clerkClient } from "@clerk/nextjs/server";
import { redirect } from "next/navigation";
import { parseAsArrayOf, parseAsInteger, parseAsString } from "nuqs/server";

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
      }`,
    );
    throw error;
  }
};

export type SearchParams = {
  events?: string | string[];
  users?: string | string[];
  rootKeys?: string | string[];
  startTime?: string | string[];
  endTime?: string | string[];
  cursor?: string | string[];
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
  cursor: string | null;
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
    cursor: bucketParser.parseServerSide(params.cursor),
  };
};

export const fetchUsersFromLogs = async (
  logs: AuditLogWithTargets[],
): Promise<Record<string, User>> => {
  try {
    // Get unique user IDs from logs
    const userIds = [...new Set(logs.filter((l) => l.actorType === "user").map((l) => l.actorId))];

    // Fetch all users in parallel
    const users = await Promise.all(
      userIds.map((userId) => clerkClient.users.getUser(userId).catch(() => null)),
    );

    // Convert array to record object
    return users.reduce(
      (acc, user) => {
        if (user) {
          acc[user.id] = user;
        }
        return acc;
      },
      {} as Record<string, User>,
    );
  } catch (error) {
    console.error("Error fetching users:", error);
    return {};
  }
};
