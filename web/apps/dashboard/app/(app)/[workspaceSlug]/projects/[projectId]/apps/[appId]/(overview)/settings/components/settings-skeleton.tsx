import { SettingCardGroup, SettingsGroups, Skeleton } from "@unkey/ui";
import { cn } from "cn";

const ROWS = [
  { titleW: "w-20", descW: "w-56", controlW: "w-64" },
  { titleW: "w-24", descW: "w-48", controlW: "w-44" },
  { titleW: "w-16", descW: "w-60", controlW: "w-52" },
];

export function SettingsSkeleton() {
  return (
    <SettingsGroups aria-busy="true">
      <output className="sr-only">Loading settings...</output>
      <SettingCardGroup>
        {ROWS.map((row) => (
          <div key={row.titleW} className="flex flex-col gap-4 px-5 py-5 lg:flex-row">
            <div className="flex shrink-0 flex-col gap-2 lg:w-2/5">
              <Skeleton className={cn("h-4 rounded", row.titleW)} />
              <Skeleton className={cn("h-3 rounded", row.descW)} />
            </div>
            <Skeleton className={cn("h-8 rounded-md", row.controlW)} />
          </div>
        ))}
      </SettingCardGroup>
    </SettingsGroups>
  );
}
