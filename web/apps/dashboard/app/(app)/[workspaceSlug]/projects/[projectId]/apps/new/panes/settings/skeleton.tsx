import { Skeleton } from "@unkey/ui";

export function SettingsFormSkeleton() {
  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-col divide-y divide-grayA-4 rounded-lg border bg-raised">
        {["w-28", "w-16"].map((width) => (
          <div key={width} className="flex flex-col gap-3 px-5 py-5">
            <Skeleton className={`h-3.5 rounded ${width}`} />
            <Skeleton className="h-8 w-full rounded-lg" />
          </div>
        ))}
      </div>
      <Skeleton className="h-9 w-full rounded-lg" />
    </div>
  );
}
