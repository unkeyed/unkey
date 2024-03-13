import { getRatelimitsDaily, getRatelimitsHourly, getRatelimitsMinutely } from "@/lib/tinybird";
import { Clock, User } from "lucide-react";
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
  const end = now.setUTCHours(now.getUTCHours() + 1, 0, 0, 0);
  const intervalMs = 1000 * 60 * 60 * 24;

  const history = await getRatelimitsHourly({
    workspaceId: workspace.id,
    namespaceId: namespace.id,
    start: end - intervalMs,
    end,
  });

  const totalRequests = history.data.reduce((sum, d) => (sum += d.total), 0);
  const totalSeconds = Math.floor(
    ((history.data.at(-1)?.time ?? 0) - (history.data.at(0)?.time ?? 0)) / 1000,
  );
  const rps = totalSeconds === 0 ? 0 : totalRequests / totalSeconds;

  const data = history.data.flatMap((d) => ({
    time: d.time,
    values: {
      success: d.success,
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
          Last request{" "}
          <time dateTime="2024-03-11T19:38:06.192Z" title="20:38:06 CET">
            2 minutes ago
          </time>
        </div>
      </div>
    </div>
  );
};
