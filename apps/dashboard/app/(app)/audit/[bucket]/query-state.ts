import { parseAsArrayOf, parseAsInteger, parseAsString, useQueryStates } from "nuqs";

export type Cursor = {
  time: number;
  id: string;
};

export type AuditLogQueryParams = {
  before: number | null;
  events: string[];
  users: string[];
  rootKeys: string[];
  bucket: string | null;
  cursorTime: number | null;
  cursorId: string | null;
};

export const auditLogParamsPayload = {
  before: parseAsInteger,
  bucket: parseAsString,
  events: parseAsArrayOf(parseAsString).withDefault([]),
  users: parseAsArrayOf(parseAsString).withDefault([]),
  rootKeys: parseAsArrayOf(parseAsString).withDefault([]),
  cursorTime: parseAsInteger,
  cursorId: parseAsString,
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
