import { StackedColumnChart } from "@/components/dashboard/charts";
import { Separator } from "@/components/ui/separator";
import { getAllSemanticCacheLogs, getSemanticCachesDaily } from "@/lib/tinybird";

interface DataEntry {
  x: string;
  y: number;
  category: string;
}

const tokenCostMap = {
  "gpt-4o": { cost: 15 / 1_000_000, tps: 63.32 },
  "gpt-4-turbo": { cost: 10 / 1_000_000, tps: 35.68 },
  "gpt-4": { cost: 30 / 1_000_000, tps: 35.68 },
  "gpt-3.5-turbo-0125": { cost: 0.5 / 1_000_000, tps: 67.84 },
} as { [key: string]: { cost: number; tps: number } };

export default async function SemanticCacheAnalyticsPage() {
  // const { data } = await getAllSemanticCacheLogs({ limit: 100 });
  const { data } = await getSemanticCachesDaily({
    start: 1717427300659,
    end: 1717427905459,
    gatewayId: "test",
    workspaceId: "test",
  });

  console.info({ data });

  const tokenCost = data.reduce((acc, log) => acc + tokenCostMap[log.model].cost * log.tokens, 0);
  const tokens = data.reduce((acc, log) => acc + log.tokens, 0);
  const timeSaved = data.reduce((acc, log) => acc + log.tokens / tokenCostMap[log.model].tps, 0);

  console.info({ tokenCost, tokens, timeSaved });

  const transformedData = data.map((log) => {
    const isCacheHit = log.cache > 0;
    return {
      x: log.time,
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

  return (
    <div>
      <div className="py-4 text-gray-200">
        <p>{timeSaved.toFixed(5)} seconds saved</p>
        <p>${tokenCost.toFixed(5)} saved in API costs</p>
        <p>{tokens} tokens served from cache</p>
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
