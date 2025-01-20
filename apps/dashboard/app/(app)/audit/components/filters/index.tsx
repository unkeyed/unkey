import { DEFAULT_BUCKET_NAME } from "@/lib/trpc/routers/audit/fetch";
import type { auditLogBucket, workspaces } from "@unkey/db/src/schema";
import { unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import { Button } from "@unkey/ui";
import { Suspense } from "react";
import type { ParsedParams } from "../../actions";
import { BucketSelect } from "./bucket-select";
import { ClearButton } from "./clear-button";
import { DatePickerWithRange } from "./datepicker-with-range";
import { Filter } from "./filter";
import { RootKeyFilter } from "./root-key-filter";
import { UserFilter } from "./user-filter";

export type SelectWorkspace = typeof workspaces.$inferSelect & {
  auditLogBuckets: Pick<typeof auditLogBucket.$inferSelect, "id" | "name">[];
};

export const Filters = ({
  selectedBucketName,
  workspace,
  parsedParams,
}: {
  selectedBucketName: string;
  workspace: SelectWorkspace;
  parsedParams: ParsedParams;
}) => {
  return (
    <div className="flex items-center justify-start gap-2 mb-4">
      <BucketSelect buckets={workspace.auditLogBuckets} />
      <Filter
        param="events"
        title="Events"
        options={
          selectedBucketName === DEFAULT_BUCKET_NAME
            ? Object.values(unkeyAuditLogEvents.Values).map((value) => ({
                value,
                label: value,
              }))
            : [
                {
                  value: "ratelimit.success",
                  label: "Ratelimit success",
                },
                { value: "ratelimit.denied", label: "Ratelimit denied" },
              ]
        }
      />
      {selectedBucketName === DEFAULT_BUCKET_NAME ? (
        <Suspense fallback={<Filter param="users" title="Users" options={[]} />}>
          <UserFilter tenantId={workspace.tenantId} />
        </Suspense>
      ) : null}
      <Suspense fallback={<Filter param="rootKeys" title="Root Keys" options={[]} />}>
        <RootKeyFilter workspaceId={workspace.id} />
      </Suspense>
      <Button className="bg-transparent">
        <DatePickerWithRange
          initialParams={{
            startTime: parsedParams.startTime,
            endTime: parsedParams.endTime,
          }}
        />
      </Button>
      <div className="w-full flex justify-end">
        <ClearButton />
      </div>
    </div>
  );
};
