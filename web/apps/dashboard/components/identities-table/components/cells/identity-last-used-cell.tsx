"use client";

import { trpc } from "@/lib/trpc/client";
import { LastUpdatedCell } from "@unkey/ui";

type IdentityLastUsedCellProps = {
  identityId: string;
  isSelected: boolean;
};

export const IdentityLastUsedCell = ({ identityId, isSelected }: IdentityLastUsedCellProps) => {
  const { data, isLoading } = trpc.identity.latestVerification.useQuery(
    { identityId },
    {
      trpc: {
        context: {
          skipBatch: true,
        },
      },
    },
  );

  if (isLoading) {
    return (
      <div className="px-1.5 rounded-md flex gap-2 items-center w-[140px] h-[22px] bg-grayA-3 animate-pulse">
        <div className="h-2 w-2 bg-grayA-5 rounded-full" />
        <div className="h-2 w-12 bg-grayA-5 rounded-sm" />
        <div className="h-2 w-12 bg-grayA-5 rounded-sm" />
      </div>
    );
  }

  return (
    <LastUpdatedCell isSelected={isSelected} lastUpdated={data?.lastVerificationTime ?? null} />
  );
};
