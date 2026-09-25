"use client";

import { IconCubeOutline18, IconGridOutline18, IconMagnifierOutline18 } from "@unkey/icons";
import { useEffect, useRef, useState } from "react";
import { AddMenu, ProtoCanvas, appEntries, serviceEntries } from "../proto-canvas";
import { HealthDot, HelpLine, type VariantProps, healthLine } from "../shared";

export function CommandVariant(p: VariantProps) {
  const [query, setQuery] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const target = e.target;
      if (target instanceof HTMLElement && /^(INPUT|TEXTAREA|SELECT)$/.test(target.tagName)) {
        if (e.key === "Escape") {
          setQuery("");
          inputRef.current?.blur();
        }
        return;
      }
      if (e.key === "/") {
        e.preventDefault();
        inputRef.current?.focus();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);
  return (
    <div className="mx-auto flex w-full max-w-[1280px] flex-col gap-4 px-8 py-8">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold tracking-tight text-gray-12">{p.title}</h1>
        <span className="flex items-center gap-2 text-xs text-gray-11">
          <HealthDot totals={p.totals} />
          {healthLine(p.totals, p.data.apps.length)}
        </span>
      </div>
      <div className="flex items-center gap-2">
        <label className="flex h-10 flex-1 items-center gap-2.5 rounded-xl border border-border bg-raised px-3 shadow-xs focus-within:border-gray-10 focus-within:shadow-[0_0_0_3px_var(--color-grayA-3)]">
          <IconMagnifierOutline18 className="size-4 text-gray-9" />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Filter apps, keyspaces and ratelimits"
            className="min-w-0 flex-1 bg-transparent text-[13px] text-gray-12 outline-none placeholder:text-gray-9"
          />
          <kbd className="rounded border border-border px-1.5 font-mono text-[11px] text-gray-9">
            /
          </kbd>
        </label>
        <div className="flex h-10 items-center gap-0.5 rounded-xl border border-border bg-raised px-1 shadow-xs">
          <AddMenu
            label="Add app"
            icon={<IconCubeOutline18 />}
            entries={appEntries(p.actions)}
            side="bottom"
          />
          <AddMenu
            label="Add service"
            icon={<IconGridOutline18 />}
            entries={serviceEntries(p.actions)}
            side="bottom"
          />
        </div>
      </div>
      <ProtoCanvas {...p} options={{ dock: false, query, heightClass: "h-[560px]" }} />
      <HelpLine />
    </div>
  );
}
