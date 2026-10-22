import { githubUrl } from "@/lib/github-url";
import { IconDotsOutline18 } from "@unkey/icons";
import { Button, InfoTooltip, useElapsed } from "@unkey/ui";
import { cn } from "cn";
import { AppActions } from "./app-actions";
import type { AppRowData } from "./app-row-model";

export function SourceLabel({ row, className }: { row: AppRowData; className?: string }) {
  const { app } = row;
  if (row.source !== "git") {
    const label =
      app.imageReference ?? (row.source === "image" ? "No image configured" : "Legacy app");
    return (
      <InfoTooltip content={label} asChild position={{ align: "start", side: "top" }}>
        <span className="min-w-0 truncate">{label}</span>
      </InfoTooltip>
    );
  }
  if (!app.repositoryFullName) {
    return <span className="text-gray-9">No repository connected</span>;
  }
  return (
    <LinkOrText href={githubUrl.repo(app.repositoryFullName)} className={className}>
      {app.repositoryFullName}
    </LinkOrText>
  );
}

export function AppActionsButton({ projectId, appId }: { projectId: string; appId: string }) {
  return (
    <AppActions projectId={projectId} appId={appId}>
      <Button variant="ghost" size="icon" className="shrink-0" title="App actions">
        <IconDotsOutline18 />
      </Button>
    </AppActions>
  );
}

export function DeployedAgo({ value, className }: { value: number; className?: string }) {
  return <span className={className}>{useElapsed(value, "long")}</span>;
}

export function LinkOrText({
  href,
  className,
  children,
}: {
  href: string | undefined;
  className?: string;
  children: string;
}) {
  return (
    <InfoTooltip content={children} asChild position={{ align: "start", side: "top" }}>
      {href ? (
        <a href={href} target="_blank" rel="noopener noreferrer" className={className}>
          {children}
        </a>
      ) : (
        <span className={cn(className, "hover:no-underline")}>{children}</span>
      )}
    </InfoTooltip>
  );
}
