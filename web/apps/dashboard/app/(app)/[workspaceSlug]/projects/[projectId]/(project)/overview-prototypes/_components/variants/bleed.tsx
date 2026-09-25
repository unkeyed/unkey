"use client";

import { IconCircleQuestionOutline18 } from "@unkey/icons";
import { AddMenu, ProtoCanvas } from "../proto-canvas";
import { HealthDot, type VariantProps, healthLine, helpEntries, useCopyPrompt } from "../shared";

export function BleedVariant(p: VariantProps) {
  const { copy } = useCopyPrompt();
  return (
    <div style={{ height: "calc(100dvh - 56px)" }}>
      <ProtoCanvas
        {...p}
        options={{
          frame: "bleed",
          heightClass: "h-full",
          overlay: (
            <div className="flex items-center gap-3 rounded-xl bg-raised py-2 pr-3 pl-3.5 shadow-floating">
              <span className="text-[15px] font-semibold text-gray-12">{p.title}</span>
              <span className="h-4 w-px bg-grayA-4" />
              <span className="flex items-center gap-2 text-xs text-gray-11">
                <HealthDot totals={p.totals} />
                {healthLine(p.totals, p.data.apps.length)}
              </span>
            </div>
          ),
          dockExtra: (
            <AddMenu
              label="Help"
              icon={<IconCircleQuestionOutline18 />}
              entries={helpEntries(copy)}
            />
          ),
        }}
      />
    </div>
  );
}
