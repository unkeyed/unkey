import {
  parseAsNumberLiteral,
  parseAsString,
  parseAsTimestamp,
  useQueryStates,
} from "nuqs";

export type PickKeys<T, K extends keyof T> = K;

// const METHODS = ["POST", "GET", "PUT", "DELETE"] as const;
const STATUSES = [400, 500, 200] as const;
// type Method = (typeof METHODS)[number];
type ResponseStatus = (typeof STATUSES)[number];

export type QuerySearchParams = {
  host: string;
  requestId: string;
  method: string;
  path: string;
  responseStatuses: ResponseStatus[] | ResponseStatus;
  startTime: Date;
  endTime: Date;
};

export const useLogSearchParams = () => {
  const [searchParams, setSearchParams] = useQueryStates({
    requestId: parseAsString,
    host: parseAsString,
    method: parseAsString,
    path: parseAsString,
    responseStatutes: parseAsNumberLiteral(STATUSES),
    startTime: parseAsTimestamp,
    endTime: parseAsTimestamp,
  });

  return { searchParams, setSearchParams };
};
