import { PageHeader } from "@/components/dashboard/page-header";

import { getAllSemanticCacheLogs } from "@/lib/tinybird";
import Link from "next/link";
import type { Interval } from "../apis/[apiId]/select";
import Client from "./form";

export default async function SemanticCachePage() {
  const { data } = await getAllSemanticCacheLogs({});

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
      <PageHeader
        title="Semantic Cache"
        description="Faster, cheaper LLM API calls through semantic caching"
      />
      <Client data={transformedData} />
    </div>
  );
}
