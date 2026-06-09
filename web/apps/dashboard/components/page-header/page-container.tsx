"use client";

import { PageBody } from "@unkey/ui";
import type { ReactNode } from "react";

/**
 * - `default` constrains and centers the page in a {@link PageBody}, for forms
 *   and detail pages.
 * - `full` runs the body edge to edge, for dense pages (logs, tables) that own
 *   their width. The header keeps the standard gutter and a full-width divider.
 */
type PageWidth = "default" | "full";

type PageContainerProps = {
  header: ReactNode;
  children: ReactNode;
  width?: PageWidth;
};

/**
 * Wraps a page header and body in the standard measure. Full-width pages get their
 * header gutter and divider here, once, so individual pages never restyle the
 * header — they just pass `width="full"`.
 */
export function PageContainer({ header, children, width = "default" }: PageContainerProps) {
  if (width === "full") {
    return (
      <>
        <div className="border-grayA-4 border-b px-4 lg:px-6 xl:px-10 pt-4 pb-4">{header}</div>
        {children}
      </>
    );
  }

  return (
    <PageBody className="pt-6">
      {header}
      {children}
    </PageBody>
  );
}
