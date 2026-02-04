import type { FlagCode } from "@/lib/trpc/routers/deploy/network/utils";
import { RegionFlag } from "./region-flag";

type RegionFlagsProps = {
  instances: { id: string; flagCode: FlagCode }[];
};

export function RegionFlags({ instances }: RegionFlagsProps) {
  return (
    <div className="gap-1 flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100 bg-grayA-3 p-1.5 h-[22px] rounded-md">
      {instances.map((instance) => (
        <RegionFlag key={instance.id} flagCode={instance.flagCode} size="xs" shape="rounded" />
      ))}
    </div>
  );
}
