import { cn } from "@/lib/utils";
import { Card } from "../../components/card";

function Bar({ className }: { className?: string }) {
  return <div className={cn("bg-grayA-3 rounded animate-pulse", className)} />;
}

const METADATA_CELLS = ["status", "region", "resources", "instances", "source", "created"];

export function ProductionDeploymentCardSkeleton() {
  return (
    <Card className="flex flex-col">
      <div className="flex items-center justify-between gap-4 px-4 py-3 border-b border-gray-4">
        <Bar className="h-4 w-40" />
        <div className="flex items-center gap-2">
          <Bar className="h-7 w-16 rounded-md" />
          <Bar className="h-7 w-24 rounded-md" />
          <Bar className="h-7 w-7 rounded-md" />
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2">
        <div className="p-4 md:border-r border-gray-4 flex flex-col gap-3">
          <div className="flex items-baseline justify-between">
            <Bar className="h-7 w-24" />
            <Bar className="h-4 w-12" />
          </div>
          <Bar className="h-[120px] w-full rounded-md" />
          <div className="flex items-center gap-4">
            <Bar className="h-3 w-20" />
            <Bar className="h-3 w-16" />
          </div>
        </div>

        <div className="p-4">
          <div className="grid grid-cols-2 gap-y-4 gap-x-6">
            {METADATA_CELLS.map((cell) => (
              <div key={cell} className="flex flex-col gap-1.5">
                <Bar className="h-3 w-16" />
                <Bar className="h-4 w-24" />
              </div>
            ))}
          </div>
        </div>
      </div>
    </Card>
  );
}
