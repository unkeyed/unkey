import { db } from "@/lib/db";
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
