"use client";

import { WorkspaceSwitcher } from "@/components/navigation/sidebar/team-switcher";

/**
 * Variant-shared: workspace switcher pinned at the top of the sidebar.
 * Edge-to-edge button with a single bottom border — arbitrary variants
 * strip the inner trigger's border / rounding / fixed height so it reads
 * as one big clickable row.
 */
export function WorkspaceSwitcherTop() {
  return (
    <div className="w-full border-b border-grayA-4 [&_button]:!h-12 [&_button]:!w-full [&_button]:!rounded-none [&_button]:!border-0 [&_button]:!px-4">
      <WorkspaceSwitcher />
    </div>
  );
}
