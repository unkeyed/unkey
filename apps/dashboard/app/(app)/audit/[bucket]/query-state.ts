import { parseAsArrayOf, parseAsInteger, parseAsString, useQueryStates } from "nuqs";

export const auditLogParamsPayload = {
  bucket: parseAsString,
  events: parseAsArrayOf(parseAsString).withDefault([]),
  users: parseAsArrayOf(parseAsString).withDefault([]),
  rootKeys: parseAsArrayOf(parseAsString).withDefault([]),
  cursor: parseAsString,
  startTime: parseAsInteger,
  endTime: parseAsInteger,
};

export const useAuditLogParams = () => {
  const [searchParams, setSearchParams] = useQueryStates(auditLogParamsPayload);

  const setCursor = (cursor?: string) => {
    setSearchParams({
      cursor,
    });
  };

  return {
    searchParams,
    setSearchParams,
    setCursor,
  };
};
