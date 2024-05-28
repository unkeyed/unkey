import { StackedColumnChart } from "@/components/dashboard/charts";
import { getAllSemanticCacheLogs } from "@/lib/tinybird";

export default async function SemanticCacheAnalyticsPage() {
  const { data } = await getAllSemanticCacheLogs({ limit: 10 });
  const transformedData = data.map((log) => {
    const isCacheHit = log.cache > 0;
    return {
      x: log.timestamp,
      y: isCacheHit ? 1 : 0, // Assuming cache > 0 indicates a cache hit
      category: isCacheHit ? "cache hit" : "cache miss",
    };
  });
  return (
    <div>
      <StackedColumnChart colors={["primary", "warn", "danger"]} data={transformedData} />
    </div>
  );
}
