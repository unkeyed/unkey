"use client";

import type { GuardMode } from "@/hooks/prevent-leave-model";
import { usePreventLeave } from "@/hooks/use-prevent-leave";
import {
  type DiscardChangesCopy,
  DiscardChangesDialog,
  UnsavedChangesScope,
  useUnsavedChanges,
} from "@unkey/ui";
import type { Route } from "next";
import { useRouter } from "next/navigation";
import { type PropsWithChildren, createContext, use } from "react";

type LeaveGuardActions = {
  navigate: (href: Route) => void;
  bypass: () => void;
};

const LeaveGuardContext = createContext<LeaveGuardActions | null>(null);

export type LeaveConfirm = {
  copy: DiscardChangesCopy;
  loading: boolean;
  beforeLeave: () => Promise<void>;
};

export function LeaveGuard({
  fallbackHref,
  mode = "off",
  confirm,
  children,
}: PropsWithChildren<{ fallbackHref: Route; mode?: GuardMode; confirm?: LeaveConfirm }>) {
  const { isDirty, report } = useUnsavedChanges();
  const { prompt, navigate, bypass } = usePreventLeave(
    isDirty && mode === "off" ? "confirm" : mode,
    isDirty,
    fallbackHref,
  );
  const onDiscard = confirm
    ? () => confirm.beforeLeave().then(prompt.onDiscard, noop)
    : prompt.onDiscard;
  return (
    <UnsavedChangesScope report={report}>
      <LeaveGuardContext value={{ navigate, bypass }}>{children}</LeaveGuardContext>
      <DiscardChangesDialog
        {...prompt}
        onDiscard={onDiscard}
        copy={confirm?.copy}
        loading={confirm?.loading}
      />
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
