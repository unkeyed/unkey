"use client";

/**
 * The debugger's REPL prompt: a query bar that reads like a sentence.
 * "Can {principal} do {action} on {resource}?" Enter or Check submits.
 */

import { ChevronDown } from "@unkey/icons";
import {
  Button,
  Input,
  Popover,
  PopoverContent,
  PopoverTrigger,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  cn,
} from "@unkey/ui";
import { useMemo, useState } from "react";
import { actionsForPath } from "../lib/catalog";
import { ALL_RESOURCES, type ConcreteResource, type MockPrincipal } from "../lib/mock-data";
import type { DebuggerQuery } from "./analysis";

const RESOURCE_LIST_LIMIT = 40;

export function QueryBar({
  principals,
  query,
  onChange,
  onCheck,
}: {
  principals: MockPrincipal[];
  query: DebuggerQuery;
  onChange: (query: DebuggerQuery) => void;
  onCheck: () => void;
}) {
  const principal = principals.find((p) => p.id === query.principalID) ?? null;
  const actions = useMemo(() => actionsForPath(query.resourcePath), [query.resourcePath]);

  const selectResource = (path: string) => {
    // The action list repopulates from the new resource type; keep the action
    // when it still applies, otherwise fall back to the first valid one.
    const nextActions = actionsForPath(path);
    const action = nextActions.some((a) => a.action === query.action)
      ? query.action
      : (nextActions[0]?.action ?? "");
    onChange({ ...query, resourcePath: path, action });
  };

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        onCheck();
      }}
      className="flex flex-wrap items-center gap-2 rounded-lg border border-grayA-4 bg-grayA-2 p-3"
    >
      <span className="text-sm text-gray-10">Can</span>

      <Select
        value={query.principalID}
        onValueChange={(value) => {
          if (typeof value === "string") {
            onChange({ ...query, principalID: value });
          }
        }}
      >
        <SelectTrigger
          aria-label="Principal"
          wrapperClassName="w-full sm:w-52 shrink-0"
          className="bg-gray-1"
        >
          {principal ? (
            <span className="truncate text-gray-12">{principal.name}</span>
          ) : (
            <span className="text-grayA-8">Pick a root key</span>
          )}
        </SelectTrigger>
        <SelectContent>
          {principals.map((p) => (
            <SelectItem key={p.id} value={p.id}>
              <span className="flex flex-col items-start gap-0.5">
                <span>{p.name}</span>
                <span className="font-mono text-[11px] text-gray-9">{p.id}</span>
              </span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <span className="text-sm text-gray-10">do</span>

      <Select
        value={query.action}
        onValueChange={(value) => {
          if (typeof value === "string") {
            onChange({ ...query, action: value });
          }
        }}
      >
        <SelectTrigger
          aria-label="Action"
          wrapperClassName="w-full sm:w-56 shrink-0"
          className="bg-gray-1"
        >
          {query.action !== "" ? (
            <span className="truncate font-mono text-xs text-info-11">{query.action}</span>
          ) : (
            <span className="text-grayA-8">Pick an action</span>
          )}
        </SelectTrigger>
        <SelectContent>
          {actions.map((a) => (
            <SelectItem key={a.action} value={a.action}>
              <span className="flex flex-col items-start gap-0.5">
                <span className="font-mono text-xs">{a.action}</span>
                <span className="text-[11px] text-gray-9">{a.description}</span>
              </span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <span className="text-sm text-gray-10">on</span>

      <ResourceCombobox value={query.resourcePath} onSelect={selectResource} />

      <Button
        type="submit"
        variant="primary"
        size="lg"
        className="px-5"
        disabled={query.principalID === "" || query.action === "" || query.resourcePath === ""}
      >
        Check
      </Button>
    </form>
  );
}

function ResourceCombobox({
  value,
  onSelect,
}: {
  value: string;
  onSelect: (path: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const selected = ALL_RESOURCES.find((r) => r.path === value) ?? null;

  const filtered = useMemo(() => {
    const needle = search.trim().toLowerCase();
    if (needle === "") {
      return ALL_RESOURCES;
    }
    return ALL_RESOURCES.filter(
      (r) =>
        r.label.toLowerCase().includes(needle) ||
        r.path.toLowerCase().includes(needle) ||
        r.type.replaceAll("_", " ").includes(needle),
    );
  }, [search]);

  const pick = (resource: ConcreteResource) => {
    onSelect(resource.path);
    setOpen(false);
  };

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (next) {
          setSearch("");
        }
      }}
    >
      <PopoverTrigger
        aria-label="Resource"
        className={cn(
          "flex h-9 min-w-0 flex-1 basis-64 items-center justify-between gap-2 rounded-lg border border-gray-5 bg-gray-1 px-3 text-left text-[13px] leading-5 text-grayA-12",
          "transition-colors duration-300 hover:border-gray-8 focus:border-accent-12 focus:ring-3 focus:ring-gray-5 focus-visible:outline-hidden",
        )}
      >
        {selected ? (
          <span className="flex min-w-0 items-baseline gap-2">
            <span className="truncate text-gray-12">{selected.label}</span>
            <span className="hidden truncate font-mono text-[11px] text-gray-9 md:inline">
              {selected.path}
            </span>
          </span>
        ) : (
          <span className="text-grayA-8">Pick a resource</span>
        )}
        <ChevronDown className="shrink-0 text-gray-11" iconSize="sm-medium" />
      </PopoverTrigger>
      <PopoverContent
        align="start"
        sideOffset={6}
        className="w-[28rem] max-w-[92vw] p-0 overflow-hidden"
      >
        <div className="border-b border-grayA-3 p-2">
          <Input
            autoFocus
            value={search}
            onChange={(event) => setSearch(event.currentTarget.value)}
            placeholder="Search by name, path, or type"
            aria-label="Search resources"
            onKeyDown={(event) => {
              if (event.key === "Enter") {
                event.preventDefault();
                const first = filtered[0];
                if (first) {
                  pick(first);
                }
              }
            }}
          />
        </div>
        <ul className="max-h-72 overflow-y-auto p-1">
          {filtered.length === 0 ? (
            <li className="px-2 py-8 text-center text-xs text-gray-9">
              {`No resources match "${search}"`}
            </li>
          ) : (
            filtered.slice(0, RESOURCE_LIST_LIMIT).map((resource) => (
              <li key={resource.path}>
                <button
                  type="button"
                  onClick={() => pick(resource)}
                  className={cn(
                    "flex w-full flex-col gap-0.5 rounded-md px-2 py-1.5 text-left transition-colors hover:bg-grayA-3",
                    resource.path === value && "bg-grayA-2",
                  )}
                >
                  <span className="flex items-center gap-2">
                    <span className="text-[13px] text-gray-12">{resource.label}</span>
                    <span className="text-[10px] uppercase tracking-wide text-gray-9">
                      {resource.type.replaceAll("_", " ")}
                    </span>
                  </span>
                  <span className="truncate font-mono text-[11px] text-gray-9">
                    {resource.path}
                  </span>
                </button>
              </li>
            ))
          )}
          {filtered.length > RESOURCE_LIST_LIMIT && (
            <li className="px-2 py-1.5 text-[11px] text-gray-9">
              {filtered.length - RESOURCE_LIST_LIMIT} more matches, keep typing to narrow down
            </li>
          )}
        </ul>
      </PopoverContent>
    </Popover>
  );
}
