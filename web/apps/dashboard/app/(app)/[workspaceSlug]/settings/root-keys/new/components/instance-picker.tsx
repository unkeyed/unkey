"use client";

import {
  Multibox,
  MultiboxContent,
  MultiboxEmpty,
  MultiboxInput,
  MultiboxItem,
  MultiboxList,
  MultiboxTrigger,
  useMultiboxAnchor,
} from "@/components/ui/multibox";
import { ChevronExpandY } from "@unkey/icons";
import { Fragment } from "react";
import type { ScopeInstance } from "../hooks/use-scope-instances";
import { ALL_INSTANCES, grantsPreview, selectInstances } from "../lib/policy";

const TRIGGER_NAME_LIMIT = 2;

type InstancePickerProps = {
  noun: string;
  instances: readonly ScopeInstance[];
  isLoading: boolean;
  value: string[];
  onChange: (instances: string[]) => void;
};

export function InstancePicker({
  noun,
  instances,
  isLoading,
  value,
  onChange,
}: InstancePickerProps) {
  const anchor = useMultiboxAnchor();
  const allLabel = `All ${noun}`;
  const items = [ALL_INSTANCES, ...instances.map((instance) => instance.id)];

  const labelOf = (id: string) =>
    id === ALL_INSTANCES ? allLabel : (instances.find((i) => i.id === id)?.label ?? id);

  const names = value.includes(ALL_INSTANCES) ? [allLabel] : value.map(labelOf);
  const { shown, more } = grantsPreview(names, TRIGGER_NAME_LIMIT);

  return (
    <Multibox
      items={items}
      value={value}
      onValueChange={(next) => onChange(selectInstances(value, next))}
      itemToStringLabel={labelOf}
    >
      <div ref={anchor}>
        <MultiboxTrigger
          aria-label={`Select ${noun}`}
          className="static right-auto top-auto flex h-9 w-full translate-y-0 items-center gap-2 rounded-lg border border-gray-5 bg-gray-2 px-3 text-[13px] leading-5 text-grayA-12 transition-colors duration-300 hover:border-gray-8 dark:bg-black"
        >
          <span className="truncate">
            {names.length === 0 ? (
              <span className="text-grayA-8">Select {noun}…</span>
            ) : (
              shown.join(", ")
            )}
          </span>
          {more > 0 ? (
            <span className="rounded-md border border-grayA-4 bg-grayA-3 px-1.5 py-0.5 text-xs text-gray-11">
              +{more}
            </span>
          ) : null}
          <ChevronExpandY iconSize="md-medium" className="ml-auto shrink-0 text-grayA-9" />
        </MultiboxTrigger>
      </div>
      <MultiboxContent anchor={anchor} className="p-0">
        <div className="border-b border-grayA-3 px-2 py-1.5">
          <MultiboxInput placeholder={`Search ${noun}…`} className="w-full text-[13px]" />
        </div>
        <div className="p-1">
          <MultiboxEmpty>{isLoading ? "Loading…" : `No ${noun} match this filter.`}</MultiboxEmpty>
          <MultiboxList>
            {(id: string) =>
              id === ALL_INSTANCES ? (
                <Fragment key={id}>
                  <MultiboxItem value={id}>{allLabel}</MultiboxItem>
                  {instances.length > 0 ? <div className="my-1 h-px bg-grayA-3" /> : null}
                </Fragment>
              ) : (
                <MultiboxItem key={id} value={id}>
                  <span className="truncate">{labelOf(id)}</span>
                  <span className="truncate text-xs text-gray-9">
                    {instances.find((i) => i.id === id)?.hint}
                  </span>
                </MultiboxItem>
              )
            }
          </MultiboxList>
        </div>
      </MultiboxContent>
    </Multibox>
  );
}
