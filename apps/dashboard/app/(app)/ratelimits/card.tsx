import { clickhouse } from "@/lib/clickhouse";
import ms from "ms";
import { Sparkline } from "./sparkline";

type Props = {
  workspace: {
    id: string;
  };
  namespace: {
    id: string;
    name: string;
  };
};

export const RatelimitCard = async ({ workspace, namespace }: Props) => {
  const now = new Date();
  const end = now.setUTCMinutes(now.getUTCMinutes() + 1, 0, 0);
  const intervalMs = 1000 * 60 * 60;

  const [history, lastUsed] = await Promise.all([
    clickhouse.ratelimits.timeseries
      .perMinute({
        identifiers: [],
        workspaceId: workspace.id,
        namespaceId: namespace.id,
        startTime: end - intervalMs,
        endTime: end,
      })
      .then((res) => res.val ?? []),
    clickhouse.ratelimits
      .latest({
        workspaceId: workspace.id,
        namespaceId: namespace.id,
        limit: 1,
      })
      .then((res) => res.val?.at(0)?.time),
  ]);

  const totalRequests = history.reduce((sum, d) => sum + d.y.total, 0);
  const totalSeconds = Math.floor(((history.at(-1)?.x ?? 0) - (history.at(0)?.x ?? 0)) / 1000);
  const rps = totalSeconds === 0 ? 0 : totalRequests / totalSeconds;

  const data = history.map((d) => ({
    time: d.x,
    values: {
      passed: d.y.passed,
      total: d.y.total,
    },
  }));

  return (
    <div className="relative overflow-hidden duration-500 border rounded-lg border-border hover:border-primary/50 group">
      <div className="h-24">
        <Sparkline data={data} />
      </div>
      <div className="px-4 pb-4">
        <div className="flex justify-between space-x-2">
          <h2 className="text-content font-semibold">{namespace.name}</h2>
          <span className="font-mono text-xs text-content-subtle">
            ~{rps.toFixed(2)} requests/s
          </span>
        </div>
        <div className="text-xs text-content-subtle">
          {lastUsed ? (
            <>
              Last request{" "}
              <time
                dateTime={new Date(lastUsed).toISOString()}
                title={new Date(lastUsed).toLocaleTimeString()}
              >
                {ms(Date.now() - lastUsed)} ago
              </time>
            </>
          ) : (
            "Never used"
          )}
        </div>
      </div>
    </div>
  );
};
