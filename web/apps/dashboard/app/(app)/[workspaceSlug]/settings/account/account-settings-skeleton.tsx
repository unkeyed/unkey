import { Skeleton } from "@unkey/ui";

export function AccountSettingsSkeleton() {
  return (
    <div className="flex flex-col gap-8">
      <output aria-live="polite" className="sr-only">
        Loading account settings...
      </output>

      <section
        aria-busy="true"
        aria-labelledby="profile-loading-heading"
        className="flex flex-col gap-3"
      >
        <h2 id="profile-loading-heading" className="m-0 text-lg font-medium">
          Profile
        </h2>
        <div aria-hidden="true" className="overflow-hidden rounded-lg border border-grayA-4">
          <SkeletonRow value="avatar" />
          <SkeletonRow action />
          <SkeletonRow action />
          <SkeletonRow />
        </div>
      </section>

      <section
        aria-busy="true"
        aria-labelledby="security-loading-heading"
        className="flex flex-col gap-3"
      >
        <div className="flex flex-col gap-1">
          <h2 id="security-loading-heading" className="m-0 text-lg font-medium">
            Security
          </h2>
          <p className="m-0 text-sm text-gray-11">
            Enroll in MFA here even when your organization does not require it.
          </p>
        </div>
        <div
          aria-hidden="true"
          className="flex min-h-20 items-center gap-4 rounded-lg border border-grayA-4 px-4 py-3"
        >
          <Skeleton className="size-10 shrink-0 rounded-md" />
          <div className="flex min-w-0 flex-1 flex-col gap-2">
            <Skeleton className="h-3.5 w-48 max-w-full" />
            <Skeleton className="h-3 w-64 max-w-full" />
          </div>
          <Skeleton className="hidden h-8 w-48 shrink-0 sm:block" />
        </div>
      </section>
    </div>
  );
}

function SkeletonRow({
  action = false,
  value = "text",
}: {
  action?: boolean;
  value?: "avatar" | "text";
}) {
  return (
    <div className="flex min-h-16 items-center gap-4 border-grayA-4 border-b px-4 py-3 last:border-b-0">
      <Skeleton className="h-3.5 w-32 max-w-[30%] shrink-0" />
      {value === "avatar" ? (
        <Skeleton className="size-10 rounded-md" />
      ) : (
        <Skeleton className="h-3.5 w-48 max-w-[40%]" />
      )}
      {action ? <Skeleton className="ml-auto h-8 w-16 shrink-0" /> : null}
    </div>
  );
}
