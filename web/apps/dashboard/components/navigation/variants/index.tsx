"use client";

import type { Sidebar } from "@/components/ui/sidebar";
import { useNavbarVariant } from "@/hooks/use-navbar-variant";
import type { Quotas, Workspace } from "@/lib/db";
import { CurrentVariant } from "./current";
import { V1aVariant } from "./v1a";
import { V1bVariant } from "./v1b";

export type VariantSidebarProps = React.ComponentProps<typeof Sidebar> & {
  workspace: Workspace & { quotas: Quotas | null };
};

/**
 * Dispatches to the active navbar variant. `current` renders the production
 * sidebar unchanged. `v1a` and `v1b` are prototype alternatives toggled via
 * the dev-only VariantSwitcher.
 */
export function NavbarVariant(props: VariantSidebarProps) {
  const { variant } = useNavbarVariant();
  switch (variant) {
    case "v1a":
      return <V1aVariant {...props} />;
    case "v1b":
      return <V1bVariant {...props} />;
    case "current":
      return <CurrentVariant {...props} />;
  }
}
