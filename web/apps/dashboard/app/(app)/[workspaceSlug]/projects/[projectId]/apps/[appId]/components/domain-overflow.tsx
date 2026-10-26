"use client";

import { plural } from "@/lib/fmt";
import { Popover, PopoverContent, PopoverTrigger } from "@unkey/ui";
import type { DisplayDomain } from "./display-domain";
import { DomainListItem } from "./domain-list-item";

export function DomainOverflow({ domains }: { domains: readonly DisplayDomain[] }) {
  if (domains.length === 0) {
    return null;
  }
  return (
    <Popover>
      <PopoverTrigger
        openOnHover
        delay={0}
        closeDelay={100}
        aria-label={`Show ${plural(domains.length, "more domain")}`}
        className="shrink-0 cursor-pointer rounded-full bg-grayA-3 px-1.5 py-0.5 font-mono text-xs tabular-nums text-gray-12 transition-colors hover:bg-grayA-4 data-popup-open:bg-grayA-4"
      >
        +{domains.length}
      </PopoverTrigger>
      <PopoverContent side="bottom" align="start" className="flex w-80 flex-col gap-2 p-3">
        <span className="text-xs font-medium text-gray-11">Assigned domains</span>
        <ul className="flex max-h-64 flex-col gap-1.5 overflow-y-auto">
          {domains.map((d) => (
            <DomainListItem key={d.id} domain={d} />
          ))}
        </ul>
      </PopoverContent>
    </Popover>
  );
}
