"use client";

import {
  type PropsWithChildren,
  createContext,
  use,
  useCallback,
  useEffect,
  useId,
  useState,
} from "react";

type ReportDirty = (id: string, dirty: boolean) => void;

const ReportDirtyContext = createContext<ReportDirty | null>(null);

export function useUnsavedChanges(): { isDirty: boolean; report: ReportDirty } {
  const [dirtyIds, setDirtyIds] = useState<ReadonlySet<string>>(() => new Set());
  const report = useCallback<ReportDirty>((id, dirty) => {
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
  return { isDirty: dirtyIds.size > 0, report };
}

export function UnsavedChangesScope({
  report,
  children,
}: PropsWithChildren<{ report: ReportDirty }>) {
  return <ReportDirtyContext.Provider value={report}>{children}</ReportDirtyContext.Provider>;
}

// No-op outside an UnsavedChangesScope, so forms can render with or without a guard.
export function useReportUnsavedChanges(dirty: boolean) {
  const report = use(ReportDirtyContext);
  const id = useId();

  useEffect(() => {
    if (!report) {
      return;
    }
    report(id, dirty);
    return () => report(id, false);
  }, [report, id, dirty]);
}
