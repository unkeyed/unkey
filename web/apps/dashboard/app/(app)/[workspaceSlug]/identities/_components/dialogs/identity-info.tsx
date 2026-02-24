import type { IdentityResponseSchema } from "@/lib/trpc/routers/identity/query";
import { Fingerprint } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import type { z } from "zod";

export const IdentityInfo = ({
  identity,
}: { identity: z.infer<typeof IdentityResponseSchema> }) => {
  return (
    <div className="flex gap-5 items-center bg-white dark:bg-black border border-grayA-5 rounded-xl py-5 pl-[18px] pr-[26px]">
      <div className="bg-grayA-5 text-gray-12 size-5 flex items-center justify-center rounded-sm">
        <Fingerprint iconSize="sm-regular" />
      </div>
      <div className="flex flex-col gap-1">
        <div className="text-accent-12 text-xs font-mono">{identity.id}</div>
        <InfoTooltip
          variant="inverted"
          content={identity.externalId}
          position={{ side: "bottom", align: "center" }}
          asChild
        >
          <div className="text-accent-9 text-xs max-w-[160px] truncate">{identity.externalId}</div>
        </InfoTooltip>
      </div>
    </div>
  );
};
