import type { Identity } from "@unkey/api/models/components";
import { IconFingerprintOutline18 } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";

export const IdentityInfo = ({ identity }: { identity: Identity }) => {
  return (
    <div className="flex gap-5 items-center bg-raised border rounded-xl py-5 pl-[18px] pr-[26px]">
      <div className="bg-grayA-5 text-gray-12 size-5 flex items-center justify-center rounded-sm">
        <IconFingerprintOutline18 className="size-3" />
      </div>
      <div className="flex flex-col gap-1">
        <div className="text-gray-12 text-xs font-mono">{identity.id}</div>
        <InfoTooltip
          variant="inverted"
          content={identity.externalId}
          position={{ side: "bottom", align: "center" }}
          asChild
        >
          <div className="text-gray-9 text-xs max-w-[160px] truncate">{identity.externalId}</div>
        </InfoTooltip>
      </div>
    </div>
  );
};
