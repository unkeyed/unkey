import { Github } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import type { ReactNode } from "react";

type RepositoryDisplayProps = {
  url: string;
  className?: string;
  showIcon?: boolean;
  children?: ReactNode;
};

const extractRepoName = (url: string): string => {
  try {
    const match = url.match(/github\.com\/([^\/]+\/[^\/]+)/);
    return match?.[1] ?? url;
  } catch {
    return url;
  }
};

export const RepoDisplay = ({
  url,
  className = "",
  showIcon = true,
  children,
}: RepositoryDisplayProps) => {
  const repoName = extractRepoName(url);

  return (
    <InfoTooltip content={url} asChild>
      <div className={`flex items-center gap-1.5 ${className}`}>
        {showIcon && <Github size="lg-medium" className="shrink-0" />}
        {children || <span className="truncate">{repoName}</span>}
      </div>
    </InfoTooltip>
  );
};
