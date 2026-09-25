"use client";

import { ProtoCanvas } from "../proto-canvas";
import { HealthDot, HelpLine, MonoLabel, type VariantProps, healthLine, num, pct } from "../shared";

export function ReadoutVariant(p: VariantProps) {
  const t = p.totals;
  const cells: Array<{ label: string; value: string; sub: string }> = [
    { label: "Apps", value: String(p.data.apps.length), sub: healthLine(t, p.data.apps.length) },
    { label: "Keys", value: num(t.keys), sub: `${p.data.keyspaces.length} keyspaces` },
    { label: "Verified · 7d", value: num(t.verified), sub: "Key verifications" },
    { label: "Ratelimited · 7d", value: num(t.rlRequests), sub: `${pct(t.rlBlockedPct)} blocked` },
  ];
  return (
    <div className="mx-auto flex w-full max-w-[1280px] flex-col gap-6 px-8 py-8">
      <div className="flex items-end justify-between">
        <div className="flex flex-col gap-1.5">
          <MonoLabel>{p.crumb}</MonoLabel>
          <h1 className="text-[28px] font-semibold leading-none tracking-tight text-gray-12">
            {p.title}
          </h1>
        </div>
        <span className="flex items-center gap-2 font-mono text-[11px] text-gray-9">
          <span className="relative flex size-1.5">
            <span className="absolute inline-flex size-full animate-ping rounded-full bg-success-9 opacity-50 motion-reduce:hidden" />
            <span className="relative inline-flex size-1.5 rounded-full bg-success-9" />
          </span>
          live · 10s
        </span>
      </div>
      <div className="grid grid-cols-4 divide-x divide-border border-y border-border">
        {cells.map((c) => (
          <div key={c.label} className="flex flex-col gap-2 px-5 py-4 first:pl-0">
            <MonoLabel>{c.label}</MonoLabel>
            <span className="text-[26px] font-medium leading-none tabular-nums tracking-tight text-gray-12">
              {c.value}
            </span>
            <span className="truncate text-xs text-gray-9">{c.sub}</span>
          </div>
        ))}
      </div>
      <figure className="flex flex-col gap-2">
        <ProtoCanvas {...p} />
        <figcaption className="flex items-center justify-between px-1">
          <span className="flex items-center gap-4">
            <MonoLabel>Fig. 1 · topology</MonoLabel>
            <span className="flex items-center gap-1.5 text-[11px] text-gray-9">
              <span className="w-4 border-t border-gray-10" /> linked
            </span>
            <span className="flex items-center gap-1.5 text-[11px] text-gray-9">
              <span className="w-4 border-t border-dashed border-gray-8" /> no link
            </span>
            <span className="flex items-center gap-1.5 text-[11px] text-gray-9">
              <HealthDot totals={t} /> status
            </span>
          </span>
          <span className="text-[11px] text-gray-9">Hover a card to trace its keys</span>
        </figcaption>
      </figure>
      <HelpLine className="pt-2" />
    </div>
  );
}
