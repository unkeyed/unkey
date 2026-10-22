"use client";

import { BranchCombobox } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/branches/branch-combobox";
import { useBuildSource } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/hooks/use-build-source";

export function BranchField({
  value,
  onChange,
}: { value: string | null; onChange: (branch: string) => void }) {
  const { projectId, connection } = useBuildSource();
  return connection ? (
    <BranchCombobox
      label="Branch"
      projectId={projectId}
      connection={connection}
      value={value ?? connection.branch}
      onSelect={onChange}
    />
  ) : null;
}
