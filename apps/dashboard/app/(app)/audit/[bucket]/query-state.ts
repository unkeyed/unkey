import { parseAsArrayOf, parseAsInteger, parseAsString, useQueryStates } from "nuqs";

export type AuditLogQueryParams = {
  before: number | null;
  events: string[];
  users: string[];
  rootKeys: string[];
  bucket: string | null;
};

export const auditLogParamsPayload = {
  before: parseAsInteger,
  bucket: parseAsString,
  events: parseAsArrayOf(parseAsString).withDefault([]),
  users: parseAsArrayOf(parseAsString).withDefault([]),
  rootKeys: parseAsArrayOf(parseAsString).withDefault([]),
};

export const useAuditLogParams = () => {
  const [searchParams, setSearchParams] = useQueryStates(auditLogParamsPayload);
  return { searchParams, setSearchParams };
};
