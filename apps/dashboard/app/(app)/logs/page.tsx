"use server";

import { generateMockLogs } from "./data";
import LogsPage from "./logs-page";
import {
  createSearchParamsCache,
  parseAsString,
  parseAsTimestamp,
} from "nuqs/server";

const mockLogs = generateMockLogs(50);

const searchParamsCache = createSearchParamsCache({
  q: parseAsString.withDefault(""),
  startDate: parseAsTimestamp,
  endDate: parseAsTimestamp,
});

export default async function Page({
  searchParams,
}: {
  params: { slug: string };
  searchParams: Record<string, string | string[] | undefined>;
}) {
  const parsedParams = searchParamsCache.parse(searchParams);
  return <LogsPage logs={mockLogs} />;
}
