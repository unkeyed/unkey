"use client";

import type { EnvVar } from "@/lib/collections/deploy/env-vars";
import { IconNote3Outline18 } from "@unkey/icons";
import { CopyButton, InfoTooltip } from "@unkey/ui";
import { HighlightMatch } from "../shared/highlight-match";

export function EnvVarNameCell({ envVar, searchQuery }: { envVar: EnvVar; searchQuery: string }) {
  const { key: variableKey, description: note } = envVar;

  return (
    <div className="flex min-w-0 items-center gap-1.5">
      <span className="min-w-0 truncate font-mono text-sm text-gray-12">
        <HighlightMatch text={variableKey} query={searchQuery} />
      </span>
      {note && (
        <InfoTooltip content={note} position={{ side: "top" }}>
          <span className="relative shrink-0 text-gray-10">
            <IconNote3Outline18 className="size-3.5" />
          </span>
        </InfoTooltip>
      )}
      <CopyButton
        value={variableKey}
        variant="ghost"
        className="relative size-5 shrink-0 text-gray-10 opacity-0 group-hover/row:opacity-100 focus-visible:opacity-100 [&_svg]:size-3"
      />
    </div>
  );
}
