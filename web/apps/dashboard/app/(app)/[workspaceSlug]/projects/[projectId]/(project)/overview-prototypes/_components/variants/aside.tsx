"use client";

import { IconArrowUpRightOutline12 } from "@unkey/icons";
import { ProtoCanvas } from "../proto-canvas";
import {
  HELP_LINKS,
  HealthDot,
  MonoLabel,
  type VariantProps,
  healthLine,
  num,
  pct,
  useCopyPrompt,
} from "../shared";

export function AsideVariant(p: VariantProps) {
  const { copied, copy } = useCopyPrompt();
  const t = p.totals;
  const stats: Array<[string, string]> = [
    ["Apps", String(p.data.apps.length)],
    ["Keyspaces", String(p.data.keyspaces.length)],
    ["Keys", num(t.keys)],
    ["Verified · 7d", num(t.verified)],
    ["Ratelimits", String(p.data.ratelimits.length)],
    ["Blocked · 7d", pct(t.rlBlockedPct)],
  ];
  return (
    <div className="flex min-h-0" style={{ height: "calc(100dvh - 56px)" }}>
      <aside className="flex w-[320px] shrink-0 flex-col gap-8 overflow-y-auto border-r border-border px-8 py-10">
        <div className="flex flex-col gap-2">
          <MonoLabel>{p.crumb}</MonoLabel>
          <h1 className="text-2xl font-semibold tracking-tight text-gray-12">{p.title}</h1>
          <span className="flex items-center gap-2 text-xs text-gray-11">
            <HealthDot totals={t} />
            {healthLine(t, p.data.apps.length)}
          </span>
        </div>
        <dl className="grid grid-cols-2 gap-x-4 gap-y-5">
          {stats.map(([k, v]) => (
            <div key={k} className="flex flex-col gap-1">
              <dt>
                <MonoLabel>{k}</MonoLabel>
              </dt>
              <dd className="text-lg font-medium tabular-nums text-gray-12">{v}</dd>
            </div>
          ))}
        </dl>
        <div className="mt-auto flex flex-col gap-1">
          <MonoLabel className="mb-1">Help</MonoLabel>
          <button
            type="button"
            onClick={copy}
            className="flex items-center justify-between rounded-md py-1 text-left text-[13px] text-gray-11 hover:text-gray-12"
          >
            {copied ? "Prompt copied" : "Set up with your agent"}
            <span className="font-mono text-[10px] text-gray-9">copy</span>
          </button>
          {HELP_LINKS.map((l) => (
            <a
              key={l.title}
              href={l.href}
              target="_blank"
              rel="noreferrer"
              className="flex items-center justify-between py-1 text-[13px] text-gray-11 hover:text-gray-12"
            >
              {l.title}
              <IconArrowUpRightOutline12 className="size-3 text-gray-9" />
            </a>
          ))}
        </div>
      </aside>
      <main className="min-w-0 flex-1 p-4">
        <ProtoCanvas {...p} options={{ heightClass: "h-full" }} />
      </main>
    </div>
  );
}
