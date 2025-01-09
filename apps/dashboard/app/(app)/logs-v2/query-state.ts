import {
  parseAsArrayOf,
  parseAsInteger,
  parseAsNumberLiteral,
  parseAsString,
  useQueryStates,
} from "nuqs";

export type PickKeys<T, K extends keyof T> = K;

const TIMELINE_OPTIONS = ["1h", "3h", "6h", "12h", "24h"] as const;
export const STATUSES = [400, 500, 200] as const;
export type ResponseStatus = (typeof STATUSES)[number];
export type Timeline = (typeof TIMELINE_OPTIONS)[number];

export type QuerySearchParams = {
  host: string | null;
  requestId: string | null;
  method: string | null;
  path: string | null;
  responseStatus: ResponseStatus[];
  startTime?: number | null;
  endTime?: number | null;
};

export const queryParamsPayload = {
  requestId: parseAsString,
  host: parseAsString,
  method: parseAsString,
  path: parseAsString,
  responseStatus: parseAsArrayOf(parseAsNumberLiteral(STATUSES)).withDefault([]),
  startTime: parseAsInteger,
  endTime: parseAsInteger,
};

export const useLogSearchParams = () => {
  const [searchParams, setSearchParams] = useQueryStates(queryParamsPayload);

  return { searchParams, setSearchParams };
};
