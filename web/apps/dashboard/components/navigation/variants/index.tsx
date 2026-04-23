"use client";

import type { Sidebar } from "@/components/ui/sidebar";
import { useNavbarVariant } from "@/hooks/use-navbar-variant";
import type { Quotas, Workspace } from "@/lib/db";
import { CurrentVariant } from "./current";
import { V1aVariant } from "./v1a";
import { V1bVariant } from "./v1b";
import { V2Variant } from "./v2";
import { V2bVariant } from "./v2b";
import { V3Variant } from "./v3";

export type VariantSidebarProps = React.ComponentProps<typeof Sidebar> & {
  workspace: Workspace & { quotas: Quotas | null };
};

/**
 * Dispatches to the active navbar variant. `current` renders the production
 * sidebar unchanged. `v1a`, `v1b`, `v2` are sidebar alternatives. `v3` swaps
 * the sidebar for a fixed top header (handled by the layout adjusting the
 * main content's top padding).
 */
export function NavbarVariant(props: VariantSidebarProps) {
  const { variant } = useNavbarVariant();
  switch (variant) {
    case "v1a":
      return <V1aVariant {...props} />;
    case "v1b":
      return <V1bVariant {...props} />;
    case "v2":
      return <V2Variant {...props} />;
    case "v2b":
      return <V2bVariant {...props} />;
    case "v3":
      return <V3Variant />;
    case "current":
      return <CurrentVariant {...props} />;
  }
}
