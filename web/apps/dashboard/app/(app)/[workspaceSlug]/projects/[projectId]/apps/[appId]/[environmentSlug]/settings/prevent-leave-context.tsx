"use client";

import { usePreventLeave } from "@/hooks/use-prevent-leave";
import {
  type PropsWithChildren,
  createContext,
  useCallback,
  useContext,
  useEffect,
  useId,
  useMemo,
  useState,
} from "react";

type LeaveGuard = {
  bypass: () => void;
  reportDirty: (id: string, dirty: boolean) => void;
};

const LeaveGuardContext = createContext<LeaveGuard | null>(null);

/** Arms "Leave site?" only while some setting below it has unsaved changes. */
export function PreventLeaveProvider({ children }: PropsWithChildren) {
  const [dirtyIds, setDirtyIds] = useState<ReadonlySet<string>>(() => new Set());
  const { bypass } = usePreventLeave(dirtyIds.size > 0, { interceptBack: false });

  const reportDirty = useCallback((id: string, dirty: boolean) => {
    setDirtyIds((prev) => {
      if (prev.has(id) === dirty) {
        return prev;
      }
      const next = new Set(prev);
      if (dirty) {
        next.add(id);
      } else {
        next.delete(id);
      }
      return next;
    });
  }, []);

  const value = useMemo(() => ({ bypass, reportDirty }), [bypass, reportDirty]);
  return <LeaveGuardContext.Provider value={value}>{children}</LeaveGuardContext.Provider>;
}

// Both are no-ops outside the settings layout: onboarding renders the same
// cards with no leave guard.
export function useReportUnsavedChanges(dirty: boolean) {
  const guard = useContext(LeaveGuardContext);
  const id = useId();

  useEffect(() => {
    if (!guard) {
      return;
    }
    guard.reportDirty(id, dirty);
    return () => guard.reportDirty(id, false);
  }, [guard, id, dirty]);
}

export function usePreventLeaveBypass(): (() => void) | undefined {
  return useContext(LeaveGuardContext)?.bypass;
}
