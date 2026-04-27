"use client";

import { CopyButton } from "@unkey/ui";
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

export function DottedLink({
  href,
  copyValue,
  external,
  className = DEFAULT_LINK_CLASS,
  children,
}: DottedLinkProps) {
  return (
    <div className="flex items-center gap-2">
      {external ? (
        <a href={href} target="_blank" rel="noopener noreferrer" className={className}>
          {children}
        </a>
      ) : (
        <Link href={href} target="_blank" rel="noopener noreferrer" className={className}>
          {children}
        </Link>
      )}
      {copyValue && <CopyButton value={copyValue} variant="ghost" className="h-4 w-4" />}
    </div>
  );
}
