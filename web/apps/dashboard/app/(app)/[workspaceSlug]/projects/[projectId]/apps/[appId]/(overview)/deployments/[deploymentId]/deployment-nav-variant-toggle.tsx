"use client";

import { cn } from "@unkey/ui/src/lib/utils";
import { type DeploymentNavVariant, useDeploymentNavVariant } from "./use-deployment-nav-variant";

const OPTIONS: { value: DeploymentNavVariant; label: string }[] = [
  { value: "breadcrumb", label: "Tabs" },
  { value: "sidebar", label: "Sidebar" },
  { value: "crumb", label: "Crumb" },
];

/** Temporary floating toggle for trying both deployment nav layouts. */
export function DeploymentNavVariantToggle() {
  const [variant, setVariant] = useDeploymentNavVariant();

  return (
    <div className="fixed bottom-4 right-4 z-50 flex items-center gap-1 rounded-full border border-grayA-4 bg-white p-1 shadow-md dark:bg-black">
      <span className="px-2 text-[11px] font-medium text-gray-9">nav</span>
      {OPTIONS.map((option) => (
        <button
          key={option.value}
          type="button"
          onClick={() => setVariant(option.value)}
          className={cn(
            "rounded-full px-3 py-1 text-xs font-medium transition-colors",
            variant === option.value
              ? "bg-accent-12 text-gray-1"
              : "text-gray-11 hover:text-accent-12",
          )}
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}
