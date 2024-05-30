import { StackedColumnChart } from "@/components/dashboard/charts";
import { Separator } from "@/components/ui/separator";
import { getAllSemanticCacheLogs } from "@/lib/tinybird";

interface DataEntry {
  x: string;
  y: number;
  category: string;
}

export default async function SemanticCacheAnalyticsPage() {
  const { data } = await getAllSemanticCacheLogs({ limit: 10 });
  const transformedData = data.map((log) => {
    const isCacheHit = log.cache > 0;
    return {
      x: log.timestamp,
      y: isCacheHit ? 1 : 0,
      category: isCacheHit ? "cache hit" : "cache miss",
    };
  });

  const dailyCounts: { [key: string]: { [key: string]: number } } = {};

  transformedData.forEach((entry) => {
    const date = new Date(entry.x).toISOString().split("T")[0];
    const category = entry.category;
    if (!dailyCounts[date]) {
      dailyCounts[date] = {};
    }
    if (!dailyCounts[date][category]) {
      dailyCounts[date][category] = 0;
    }
    dailyCounts[date][category] += 1;
  });

  // Convert the aggregated data back into the specified format
  const finalData: DataEntry[] = [];
  for (const date in dailyCounts) {
    for (const category in dailyCounts[date]) {
      finalData.push({
        x: date,
        y: dailyCounts[date][category],
        category: category,
      });
    }
  }

  console.info(finalData);

  return (
    <div>
      <div className="py-4">
        Metrics
        <p>Time saved: </p>
        <p>$ saved: </p>
        <p>Tokens served from cache: </p>
      </div>
      <Separator />
      <StackedColumnChart
        colors={["primary", "warn", "danger"]}
        data={finalData}
        timeGranularity="day"
      />
    </div>
  );
}
