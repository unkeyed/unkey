"use client";

import { usePreventLeave } from "@/hooks/use-prevent-leave";
import { DiscardChangesDialog, UnsavedChangesScope, useUnsavedChanges } from "@unkey/ui";
import type { Route } from "next";
import { useRouter } from "next/navigation";
import { type PropsWithChildren, createContext, use } from "react";

type LeaveGuardActions = {
  navigate: (href: Route) => void;
  bypass: () => void;
};

const LeaveGuardContext = createContext<LeaveGuardActions | null>(null);

export function LeaveGuard({ fallbackHref, children }: PropsWithChildren<{ fallbackHref: Route }>) {
  const { isDirty, report } = useUnsavedChanges();
  const { prompt, navigate, bypass } = usePreventLeave(isDirty, fallbackHref);
  return (
    <UnsavedChangesScope report={report}>
      <LeaveGuardContext value={{ navigate, bypass }}>{children}</LeaveGuardContext>
      <DiscardChangesDialog {...prompt} />
    </UnsavedChangesScope>
  );
}

export function useGuardedNavigate(): (href: Route) => void {
  const guard = use(LeaveGuardContext);
  const router = useRouter();
  return guard?.navigate ?? ((href) => router.push(href));
}

export function useLeaveBypass(): () => void {
  return use(LeaveGuardContext)?.bypass ?? noop;
}

function noop() {}
