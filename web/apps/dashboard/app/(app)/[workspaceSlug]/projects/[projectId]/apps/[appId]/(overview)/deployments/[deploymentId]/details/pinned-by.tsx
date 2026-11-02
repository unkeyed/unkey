"use client";

import { IconCircleInfoOutline12 } from "@unkey/icons";
import { Button, Tooltip, TooltipContent, TooltipTrigger } from "@unkey/ui";
import Link from "next/link";
import type { ReactNode } from "react";
import type { PinningApp } from "../../../connections/hooks/use-deployment-connections";

const SHOWN = 2;

const PINNED_HINT = {
  one: "An app is connected to this deployment, it will continue to run until you change or delete the connection.",
  many: "Apps are connected to this deployment, it will continue to run until you change or delete the connections.",
};

export function PinningApps({
  apps,
  truncated,
}: { apps: readonly PinningApp[]; truncated: boolean }) {
  const shown = apps.slice(0, SHOWN);
  const hidden = apps.slice(SHOWN);
  const hiddenNames = overflowNames(hidden, truncated);
  return (
    <>
      {shown.map((app, index) => (
        <span key={app.id} className="flex min-w-0 items-center">
          <Link
            href={app.href}
            className="truncate underline decoration-dotted underline-offset-3 hover:decoration-solid"
          >
            {app.name}
          </Link>
          {index < shown.length - 1 && <span className="text-gray-10">,</span>}
        </span>
      ))}
      {(hidden.length > 0 || truncated) && (
        <Hint label={hiddenNames} content={hiddenNames}>
          <span className="shrink-0 text-xs text-gray-10">
            {overflowLabel(hidden.length, truncated)}
          </span>
        </Hint>
      )}
      <Hint
        label="Why this deployment is pinned"
        content={
          <span className="flex max-w-64">
            {apps.length > 1 || truncated ? PINNED_HINT.many : PINNED_HINT.one}
          </span>
        }
      >
        <span className="flex shrink-0 text-gray-9">
          <IconCircleInfoOutline12 className="size-3" />
        </span>
      </Hint>
    </>
  );
}

function overflowLabel(count: number, truncated: boolean): string {
  return count > 0 ? `+${count}${truncated ? "+" : ""} more` : "+ more";
}

function overflowNames(hidden: readonly PinningApp[], truncated: boolean): string {
  const names = hidden.map((app) => app.name);
  return (truncated ? [...names, "more apps"] : names).join(", ");
}

function Hint({
  label,
  content,
  children,
}: { label: string; content: ReactNode; children: ReactNode }) {
  return (
    <Tooltip>
      <TooltipTrigger
        aria-label={label}
        className="flex shrink-0 rounded-sm focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-gray-7"
      >
        {children}
      </TooltipTrigger>
      <TooltipContent side="top">{content}</TooltipContent>
    </Tooltip>
  );
}

export function PinningAppsError({ retry, retrying }: { retry: () => void; retrying: boolean }) {
  return (
    <>
      <span className="text-gray-10">Couldn't load</span>
      <Button variant="ghost" size="sm" onClick={retry} loading={retrying}>
        Retry
      </Button>
    </>
  );
}
