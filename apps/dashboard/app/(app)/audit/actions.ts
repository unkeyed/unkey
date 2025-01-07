import { db } from "@/lib/db";
import { redirect } from "next/navigation";
import { parseAsArrayOf, parseAsInteger, parseAsString } from "nuqs/server";

export const getWorkspace = async (tenantId: string) => {
  try {
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { eq, and, isNull }) =>
        and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
      with: {
        auditLogBuckets: {
          columns: {
            id: true,
            name: true,
          },
          orderBy: (table, { asc }) => asc(table.createdAt),
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
  bucketName?: string;
};

export type ParsedParams = {
  selectedEvents: string[];
  selectedUsers: string[];
  selectedRootKeys: string[];
  startTime: number | null;
  endTime: number | null;
  bucketName: string;
  cursor: string | null;
};

export const parseFilterParams = (params: SearchParams): ParsedParams => {
  const filterParser = parseAsArrayOf(parseAsString).withDefault([]);
  const timeParser = parseAsInteger;
  const bucketParser = parseAsString;

  return {
    selectedEvents: filterParser.parseServerSide(params.events),
    selectedUsers: filterParser.parseServerSide(params.users),
    selectedRootKeys: filterParser.parseServerSide(params.rootKeys),
    startTime: timeParser.parseServerSide(params.startTime),
    endTime: timeParser.parseServerSide(params.endTime),
    bucketName: bucketParser.withDefault("unkey_mutations").parseServerSide(params.bucketName),
    cursor: bucketParser.parseServerSide(params.cursor),
  };
};
