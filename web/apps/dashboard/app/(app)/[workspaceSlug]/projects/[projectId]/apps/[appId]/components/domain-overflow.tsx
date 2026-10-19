"use client";

import { plural } from "@/lib/fmt";
import { CopyButton, Popover, PopoverContent, PopoverTrigger } from "@unkey/ui";
import type { DisplayDomain } from "./display-domain";
import { TagBadge } from "./tag-badge";

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
            <li key={d.id} className="flex min-w-0 items-center gap-2">
              <a
                href={d.url}
                target="_blank"
                rel="noopener noreferrer"
                title={d.hostname}
                className="min-w-0 flex-1 truncate text-xs leading-5 text-gray-12 transition-colors hover:text-gray-11"
              >
                {d.hostname}
              </a>
              {d.source === "custom" && <TagBadge text="Custom" />}
              <CopyButton
                value={d.url}
                variant="ghost"
                className="size-5 shrink-0"
                toastMessage={d.hostname}
              />
            </li>
          ))}
        </ul>
      </PopoverContent>
    </Popover>
  );
}
