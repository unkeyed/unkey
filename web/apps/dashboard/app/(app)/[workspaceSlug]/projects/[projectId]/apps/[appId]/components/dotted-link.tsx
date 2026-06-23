"use client";

import { CopyButton } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import type { Route } from "next";
import Link from "next/link";

const DEFAULT_LINK_CLASS =
  "text-accent-12 text-xs decoration-dotted underline underline-offset-3 transition-all font-medium";

type DottedLinkProps = {
  href: string;
  copyValue?: string;
  external?: boolean;
  className?: string;
  children: React.ReactNode;
};

export function DottedLink({ href, copyValue, external, className, children }: DottedLinkProps) {
  const linkClassName = cn(DEFAULT_LINK_CLASS, className);

  return (
    <div className="flex min-w-0 items-center gap-2">
      {external ? (
        <a href={href} target="_blank" rel="noopener noreferrer" className={linkClassName}>
          {children}
        </a>
      ) : (
        <Link
          href={href as Route}
          target="_blank"
          rel="noopener noreferrer"
          className={linkClassName}
        >
          {children}
        </Link>
      )}
      {copyValue && <CopyButton value={copyValue} variant="ghost" className="h-4 w-4" />}
    </div>
  );
}
