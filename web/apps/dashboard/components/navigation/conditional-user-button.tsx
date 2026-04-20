"use client";

import { UserButton } from "@/components/navigation/sidebar/user-button";
import { useNavbarVariant } from "@/hooks/use-navbar-variant";

/**
 * Renders the top-bar UserButton only when the active variant is `current`.
 * `v1a` and `v1b` move the UserButton into the sidebar footer, so the top bar
 * must not render a duplicate. Client-only so localStorage state is respected.
 */
export function ConditionalUserButton() {
  const { variant } = useNavbarVariant();
  if (variant !== "current") {
    return null;
  }
  return <UserButton />;
}
