import { FormField, Skeleton } from "@unkey/ui";
import type { SourceKind } from "../wizard-model";
import { ConfigBlocksSkeleton } from "./config-blocks";
import { PaneSubmit } from "./pane-actions";

function FieldSkeleton({ label }: { label: string }) {
  return <FormField label={label}>{() => <Skeleton className="h-9 rounded-md" />}</FormField>;
}

export function ConfigureSkeleton({ source }: { source: SourceKind | null }) {
  return (
    <div aria-busy="true" className="flex flex-col gap-6">
      <div className="grid grid-cols-1 gap-4 @lg:grid-cols-2">
        <FieldSkeleton label="Name" />
        {source === "git" ? <FieldSkeleton label="Branch" /> : null}
      </div>
      <ConfigBlocksSkeleton />
      <PaneSubmit loading={false} disabled onClick={() => undefined}>
        Deploy
      </PaneSubmit>
    </div>
  );
}
