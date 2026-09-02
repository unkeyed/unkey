import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { IconKey2Outline12 } from "nucleo-ui-outline-12";

export const RootKeyInfo = ({
  rootKeyDetails,
}: {
  rootKeyDetails: RootKey;
}) => {
  return (
    <div className="flex gap-5 items-center bg-white dark:bg-black border border-grayA-5 rounded-xl py-5 pl-[18px] pr-[26px]">
      <div className="bg-grayA-5 text-gray-12 size-5 flex items-center justify-center rounded-sm ">
        <IconKey2Outline12 />
      </div>
      <div className="flex flex-col gap-1">
        <div className="text-accent-12 text-[13px] font-normal">
          {rootKeyDetails.name ?? "Unnamed Root Key"}
        </div>
        <div className="text-accent-9 text-xs max-w-[160px] truncate">
          {rootKeyDetails.start}...
        </div>
      </div>
    </div>
  );
};
