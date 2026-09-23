import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { IconKey2Outline12 } from "@unkey/icons";

export const RootKeyInfo = ({
  rootKeyDetails,
}: {
  rootKeyDetails: RootKey;
}) => {
  return (
    <div className="flex gap-5 items-center bg-raised border rounded-xl py-5 pl-[18px] pr-[26px]">
      <div className="bg-grayA-5 text-gray-12 size-5 flex items-center justify-center rounded-sm ">
        <IconKey2Outline12 />
      </div>
      <div className="flex flex-col gap-1">
        <div className="text-gray-12 text-[13px] font-medium">
          {rootKeyDetails.name ?? "Unnamed Root Key"}
        </div>
        <div className="text-gray-9 text-xs max-w-[160px] truncate">{rootKeyDetails.start}...</div>
      </div>
    </div>
  );
};
