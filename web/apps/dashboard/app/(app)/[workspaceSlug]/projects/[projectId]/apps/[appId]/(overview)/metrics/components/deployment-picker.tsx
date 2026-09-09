"use client";

import { CodeCommit } from "@unkey/icons";
import {
  ComboboxContent,
  ComboboxEmpty,
  ComboboxIcon,
  ComboboxInput,
  ComboboxItem,
  ComboboxItemIndicator,
  ComboboxList,
  ComboboxRoot,
  ComboboxTrigger,
} from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useMemo, useState } from "react";
import type { DeploymentInfo } from "../hooks/use-app-metrics";
import { shortSha } from "../lib/series";

type Props = {
  deployments: DeploymentInfo[];
  colorOf: (id: string) => string;
  selected: string[];
  onChange: (ids: string[]) => void;
  inRange: (d: DeploymentInfo) => boolean;
};

// Top-line multi-select. Nothing picked means the charts show the app total
// with a marker for every deploy in range. Picking deployments splits every
// chart into one series per pick and keeps only their markers, so two
// releases can be compared side by side.
export function DeploymentPicker({ deployments, colorOf, selected, onChange, inRange }: Props) {
  const [query, setQuery] = useState("");
  const byId = useMemo(() => new Map(deployments.map((d) => [d.id, d])), [deployments]);
  const items = useMemo(() => deployments.map((d) => d.id), [deployments]);
  const picked = selected.map((id) => byId.get(id)).filter((d): d is DeploymentInfo => !!d);

  const filter = (id: string, q: string) => {
    const d = byId.get(id);
    if (!d) {
      return false;
    }
    const haystack = `${d.sha ?? ""} ${d.message ?? ""} ${d.branch} ${d.status}`.toLowerCase();
    return haystack.includes(q.trim().toLowerCase());
  };

  return (
    <div className="flex items-center gap-2">
      <ComboboxRoot<string, true>
        multiple
        items={items}
        filter={filter}
        value={selected}
        onValueChange={(next) => onChange(next)}
        inputValue={query}
        onInputValueChange={setQuery}
      >
        <ComboboxTrigger className="h-8 w-auto min-w-[200px] max-w-[520px] text-[13px] [&_svg]:size-3">
          <CodeCommit iconSize="sm-regular" className="text-gray-11 shrink-0" />
          {picked.length === 0 ? (
            <span className="text-gray-11">All deployments</span>
          ) : (
            <span className="flex items-center gap-1.5 min-w-0">
              {picked.slice(0, 3).map((d) => (
                <span
                  key={d.id}
                  className="inline-flex items-center gap-1.5 h-5 px-1.5 rounded bg-grayA-3 text-[12px] text-gray-12 max-w-[160px]"
                >
                  <span
                    className="size-1.5 rounded-full shrink-0"
                    style={{ backgroundColor: colorOf(d.id) }}
                  />
                  <span className="font-mono">{shortSha(d.sha) || d.id.slice(-7)}</span>
                </span>
              ))}
              {picked.length > 3 && (
                <span className="text-[12px] text-gray-11">+{picked.length - 3}</span>
              )}
            </span>
          )}
          <ComboboxIcon className="ml-auto" />
        </ComboboxTrigger>
        <ComboboxContent className="w-[440px]" align="start">
          <div className="p-1">
            <ComboboxInput placeholder="Search deployments…" />
          </div>
          <ComboboxEmpty>No deployments match.</ComboboxEmpty>
          <ComboboxList>
            {(id: string) => {
              const d = byId.get(id);
              if (!d) {
                return null;
              }
              const dim = !inRange(d);
              return (
                <ComboboxItem key={id} value={id} className="gap-2.5">
                  <span
                    className="size-2 rounded-full shrink-0"
                    style={{ backgroundColor: colorOf(id) }}
                  />
                  <span className="font-mono text-gray-12 shrink-0">
                    {shortSha(d.sha) || id.slice(-7)}
                  </span>
                  <span className={cn("truncate", dim ? "text-gray-10" : "text-gray-11")}>
                    {d.message ?? id}
                  </span>
                  <span className="ml-auto text-[11px] text-gray-9 tabular-nums shrink-0 pl-2">
                    {new Date(d.createdAt).toLocaleDateString(undefined, {
                      month: "short",
                      day: "numeric",
                    })}
                  </span>
                  <ComboboxItemIndicator className="ml-2" />
                </ComboboxItem>
              );
            }}
          </ComboboxList>
          {selected.length > 0 && (
            <div className="border-t border-gray-4 p-1">
              <button
                type="button"
                onClick={() => onChange([])}
                className="w-full h-7 rounded-sm text-[12px] text-gray-11 hover:bg-grayA-3 hover:text-gray-12 transition-colors"
              >
                Clear selection
              </button>
            </div>
          )}
        </ComboboxContent>
      </ComboboxRoot>
    </div>
  );
}
