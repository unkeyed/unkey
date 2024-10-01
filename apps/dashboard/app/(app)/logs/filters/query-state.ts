import {
  parseAsArrayOf,
  parseAsNumberLiteral,
  parseAsString,
  parseAsStringLiteral,
  parseAsTimestamp,
  useQueryStates,
} from "nuqs";

export type PickKeys<T, K extends keyof T> = K;

// const METHODS = ["POST", "GET", "PUT", "DELETE"] as const;
const TIMELINE_OPTIONS = ["1h", "3h", "6h", "12h", "24h"] as const;
const STATUSES = [400, 500, 200] as const;
// type Method = (typeof METHODS)[number];
export type ResponseStatus = (typeof STATUSES)[number];
export type Timeline = (typeof TIMELINE_OPTIONS)[number];

export type QuerySearchParams = {
  host: string;
  requestId: string;
  method: string;
  path: string;
  responseStatuses: ResponseStatus[] | ResponseStatus;
  startTime: Date;
  endTime: Date;
};

const RESPONSE_STATUS_SEPARATOR = ",";
export const useLogSearchParams = () => {
  const [searchParams, setSearchParams] = useQueryStates({
    requestId: parseAsString,
    host: parseAsString,
    method: parseAsString,
    path: parseAsString,
    responseStatutes: parseAsArrayOf(
      parseAsNumberLiteral(STATUSES),
      RESPONSE_STATUS_SEPARATOR
    ),
    startTime: parseAsTimestamp,
    endTime: parseAsTimestamp,
  });

  return { searchParams, setSearchParams };
};
