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

export const RatelimitCard: React.FC<Props> = async ({ workspace, namespace }) => {
  const now = new Date();
  const end = now.setUTCMinutes(now.getUTCMinutes() + 1, 0, 0);
  const intervalMs = 1000 * 60 * 60;

  const [history, lastUsed] = await Promise.all([
    clickhouse.ratelimits.perMinute({
      workspaceId: workspace.id,
      namespaceId: namespace.id,
      start: end - intervalMs,
      end,
    }),
    clickhouse.ratelimits
      .latest({ workspaceId: workspace.id, namespaceId: namespace.id })
      .then((res) => res.at(0)?.time),
  ]);

  const totalRequests = history.reduce((sum, d) => sum + d.total, 0);
  const totalSeconds = Math.floor(
    ((history.at(-1)?.time ?? 0) - (history.at(0)?.time ?? 0)) / 1000,
  );
  const rps = totalSeconds === 0 ? 0 : totalRequests / totalSeconds;

  const data = history.flatMap((d) => ({
    time: d.time,
    values: {
      passed: d.passed,
      total: d.total,
    },
  }));
  return (
    <div className="relative overflow-hidden duration-500 border rounded-lg border-border hover:border-primary/50 group ">
      <div className="h-24 ">
        <Sparkline data={data} />
      </div>
      <div className="px-4 pb-4">
        <div className="flex justify-between space-x-2">
          <h2 className=" text-contentfont-semibold">{namespace.name}</h2>
          <span className="font-mono text-xs text-content-subtle">
            ~{rps.toFixed(2)} requests/s
          </span>
        </div>

        <div className="text-xs text-content-subtle">
          {lastUsed ? (
            <>
              Last request{" "}
              <time dateTime="2024-03-11T19:38:06.192Z" title="20:38:06 CET">
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
