import { Skeleton } from "@unkey/ui";
/**
 * Placeholder shown while a log's enrichment data (limit, duration, reset, region)
 * is still being fetched from the enrichment endpoint.
 */
export const EnrichmentSkeleton = () => <Skeleton className="h-4 w-16 bg-gray-3" />;
