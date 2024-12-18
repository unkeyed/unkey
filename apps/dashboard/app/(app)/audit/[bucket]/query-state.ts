import { parseAsArrayOf, parseAsInteger, parseAsString, useQueryStates } from "nuqs";

export type Cursor = {
  time: number;
  id: string;
};

export type AuditLogQueryParams = {
  events: string[];
  users: string[];
  rootKeys: string[];
  bucket?: string;
  cursorTime: number | null;
  cursorId: string | null;
  startTime: number | null;
  endTime: number | null;
};

export const auditLogParamsPayload = {
  bucket: parseAsString,
  events: parseAsArrayOf(parseAsString).withDefault([]),
  users: parseAsArrayOf(parseAsString).withDefault([]),
  rootKeys: parseAsArrayOf(parseAsString).withDefault([]),
  cursorTime: parseAsInteger,
  cursorId: parseAsString,
  startTime: parseAsInteger,
  endTime: parseAsInteger,
};

export const useAuditLogParams = () => {
  const [searchParams, setSearchParams] = useQueryStates(auditLogParamsPayload);

  const setCursor = (cursor?: Cursor) => {
    setSearchParams({
      cursorTime: cursor?.time ?? null,
      cursorId: cursor?.id ?? null,
    });
  };

  return {
    searchParams,
    setSearchParams,
    setCursor,
  };
};
