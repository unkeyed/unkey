"use client";

import { useFlag } from "@/lib/flags/provider";
import { PageShell } from "@unkey/ui";
import type { ReactNode } from "react";

/**
 * - `default` constrains and centers the page in a {@link PageShell}, for forms
 *   and detail pages.
 * - `full` runs the body edge to edge, for dense pages (logs, tables) that own
 *   their width. The header keeps the standard gutter and a full-width divider.
 */
type PageWidth = "default" | "full";

type PageChromeProps = {
  /** Header shown when the new navigation is on, typically a `<PageHeader>`. */
  header: ReactNode;
  /** Header shown when the new navigation is off — the existing `Navbar`. */
  legacyHeader: ReactNode;
  children: ReactNode;
  /** How the page is measured. Defaults to `default`. */
  width?: PageWidth;
};

/**
 * The single fork point between the old per-page `Navbar` and the redesigned
 * `PageHeader`. Pages pass both headers and their body once; PageChrome picks a
 * tree based on the `newNavigation` flag. When that flag is fully shipped, this
 * component collapses to its new-navigation branch and the legacy headers go.
 *
 * Full-width pages get their header gutter and divider here, once, so individual
 * pages never restyle the header — they just pass `width="full"`.
 */
export function PageChrome({ header, legacyHeader, children, width = "default" }: PageChromeProps) {
  const newNavigation = useFlag("newNavigation");

  if (!newNavigation) {
    return (
      <>
        {legacyHeader}
        {children}
      </>
    );
  }

  if (width === "full") {
    return (
      <>
        <div className="border-grayA-4 border-b px-4 lg:px-6 xl:px-10 pt-4 pb-4">{header}</div>
        {children}
      </>
    );
  }

  return (
    <PageShell className="pt-6">
      {header}
      {children}
    </PageShell>
  );
}
