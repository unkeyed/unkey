"use client";

import * as React from "react";

type ReportDirty = (id: string, dirty: boolean) => void;

const ReportDirtyContext = React.createContext<ReportDirty | null>(null);

export function useUnsavedChanges(): { isDirty: boolean; report: ReportDirty } {
  const [dirtyIds, setDirtyIds] = React.useState<ReadonlySet<string>>(() => new Set());
  const report = React.useCallback<ReportDirty>((id, dirty) => {
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
}: React.PropsWithChildren<{ report: ReportDirty }>) {
  return <ReportDirtyContext.Provider value={report}>{children}</ReportDirtyContext.Provider>;
}

/** No-op outside an UnsavedChangesScope. */
export function useReportUnsavedChanges(dirty: boolean) {
  const report = React.use(ReportDirtyContext);
  const id = React.useId();

  React.useEffect(() => {
    if (!report) {
      return;
    }
    report(id, dirty);
    return () => report(id, false);
  }, [report, id, dirty]);
}
