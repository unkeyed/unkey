import { DatePickerWithRange } from "@/app/(app)/logs/components/filters/components/custom-date-filter";
import { DEFAULT_BUCKET_NAME } from "@/lib/trpc/routers/audit/fetch";
import { ratelimitNamespaces, workspaces } from "@unkey/db/src/schema";
import { unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import { Button } from "@unkey/ui";
import { Suspense } from "react";
import { BucketSelect } from "./bucket-select";
import { ClearButton } from "./clear-button";
import { Filter } from "./filter";
import { RootKeyFilter } from "./root-key-filter";
import { UserFilter } from "./user-filter";
import { ParsedParams } from "../../actions";

export type SelectWorkspace = typeof workspaces.$inferSelect & {
  ratelimitNamespaces: Pick<
    typeof ratelimitNamespaces.$inferSelect,
    "id" | "name"
  >[];
};

export const Filters = ({
  bucket,
  workspace,
  parsedParams,
}: {
  bucket: string | null;
  workspace: SelectWorkspace;
  parsedParams: ParsedParams;
}) => {
  return (
    <div className="flex items-center justify-start gap-2 mb-4">
      <BucketSelect
        selected={bucket ?? DEFAULT_BUCKET_NAME}
        ratelimitNamespaces={workspace.ratelimitNamespaces}
      />
      <Filter
        param="events"
        title="Events"
        options={
          bucket === DEFAULT_BUCKET_NAME
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
      {bucket === DEFAULT_BUCKET_NAME ? (
        <Suspense
          fallback={<Filter param="users" title="Users" options={[]} />}
        >
          <UserFilter tenantId={workspace.tenantId} />
        </Suspense>
      ) : null}
      <Suspense
        fallback={<Filter param="rootKeys" title="Root Keys" options={[]} />}
      >
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
