import { Github } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import type { ReactNode } from "react";

type RepositoryDisplayProps = {
  url: string;
  className?: string;
  showIcon?: boolean;
  children?: ReactNode;
};

export const RepoDisplay = ({
  url,
  className = "",
  showIcon = true,
  children,
}: RepositoryDisplayProps) => {
  const repoName = extractRepoName(url);
  const safeHref = isSafeHttpUrl(url) ? url : undefined;

  return (
    <InfoTooltip
      content={url}
      asChild
      position={{ side: "top", align: "center" }}
      className="z-auto"
    >
      <a
        href={safeHref}
        target="_blank"
        rel="noopener noreferrer"
        className={cn(
          "flex items-center gap-1.5 transition-all hover:underline decoration-dashed underline-offset-2",
          className,
        )}
      >
        {showIcon && <Github iconSize="lg-medium" className="shrink-0" />}
        {children || <span className="truncate">{repoName}</span>}
      </a>
    </InfoTooltip>
  );
};

const extractRepoName = (url: string): string => {
  try {
    const match = url.match(/github\.com\/([^\/]+\/[^\/]+)/);
    return match?.[1] ?? url;
  } catch {
    return url;
  }
};

const isSafeHttpUrl = (href: string): boolean => {
  try {
    const u = new URL(href);
    return u.protocol === "http:" || u.protocol === "https:";
  } catch {
    return false;
  }
};
