"use client";

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
  /** The page header, typically a `<PageHeader>`. */
  header: ReactNode;
  children: ReactNode;
  /** How the page is measured. Defaults to `default`. */
  width?: PageWidth;
};

/**
 * Wraps a page header and body in the standard shell. Full-width pages get their
 * header gutter and divider here, once, so individual pages never restyle the
 * header — they just pass `width="full"`.
 */
export function PageChrome({ header, children, width = "default" }: PageChromeProps) {
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
