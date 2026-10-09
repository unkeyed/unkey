import { IconCodeBranchOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import { Badge } from "@unkey/ui";
import { cn } from "cn";
import type { Branch } from "./branch";

export function BranchLabel({ branch, className }: { branch: Branch; className?: string }) {
  return (
    <span className={cn("flex min-w-0 items-center gap-1.5 text-xs text-gray-11", className)}>
      {match(branch)
        .with({ kind: "default" }, ({ name }) => (
          <>
            <IconCodeBranchOutline18 className="size-3 shrink-0" />
            <Badge variant="code">{name}</Badge>
          </>
        ))
        .with({ kind: "unassigned" }, () => (
          <>
            <IconCodeBranchOutline18 className="size-3 shrink-0" />
            <span className="truncate">All unassigned branches</span>
          </>
        ))
        .with({ kind: "none" }, () => <span className="text-gray-9">—</span>)
        .exhaustive()}
    </span>
  );
}
