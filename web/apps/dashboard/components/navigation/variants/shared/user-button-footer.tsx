"use client";

import { UserButton } from "@/components/navigation/sidebar/user-button";

/**
 * Variant-shared: user menu mounted at the bottom of the sidebar
 * (v1a/v1b pull it out of the top bar into this footer slot).
 * Edge-to-edge full-width button with a single top border — mirrors the
 * `WorkspaceSwitcherTop` treatment so the sidebar chrome reads as two
 * matching rails top and bottom.
 */
export function UserButtonFooter() {
  return (
    <div className="w-full border-t border-grayA-4 [&_button]:!h-12 [&_button]:!w-full [&_button]:!justify-start [&_button]:!rounded-none [&_button]:!px-4">
      <UserButton />
    </div>
  );
}
