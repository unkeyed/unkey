"use client";

import { cn } from "@/lib/utils";
import { LaunchpadPicker, useLaunchpadPicker } from "./options";
import { findVariant } from "./registry";
import { useLaunchpad } from "./use-launchpad";

export function useLaunchpadSurface() {
  const { state, update } = useLaunchpadPicker();
  const model = useLaunchpad();
  const variant = findVariant(state.variant);
  const options = {
    spark: state.spark,
    density: state.density,
    showProject: state.showProject,
    forceEmpty: state.forceEmpty,
  };

  const content = <variant.Component model={model} options={options} />;

  return {
    picker: <LaunchpadPicker state={state} update={update} />,
    rail:
      variant.placement === "rail" ? (
        <aside className={cn("flex w-full shrink-0 flex-col gap-4", variant.width ?? "lg:w-[320px]")}>
          {content}
        </aside>
      ) : null,
    band: variant.placement === "band" ? <div className="mt-8">{content}</div> : null,
  };
}
