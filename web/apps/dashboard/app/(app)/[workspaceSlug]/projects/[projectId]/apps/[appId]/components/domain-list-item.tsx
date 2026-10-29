import { CopyButton } from "@unkey/ui";
import { cn } from "cn";
import type { DisplayDomain } from "./display-domain";
import { TagBadge } from "./tag-badge";

export function DomainListItem({
  domain,
  className,
}: { domain: DisplayDomain; className?: string }) {
  return (
    <li className={cn("flex min-w-0 items-center gap-2", className)}>
      <a
        href={domain.url}
        target="_blank"
        rel="noopener noreferrer"
        title={domain.hostname}
        className="min-w-0 flex-1 truncate text-xs leading-5 text-gray-12 transition-colors hover:text-gray-11"
      >
        {domain.hostname}
      </a>
      {domain.source === "custom" && <TagBadge text="Custom" />}
      <CopyButton
        value={domain.url}
        variant="ghost"
        className="size-5 shrink-0"
        toastMessage={domain.hostname}
      />
    </li>
  );
}
