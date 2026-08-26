import { Key2 } from "@unkey/icons";

// The rotate flow reaches this from the list row (a full `RootKey`) and from
// the edit aside (a draft), so it asks only for the fields it renders.
export type RootKeySummary = {
  id: string;
  name: string | null;
  start: string;
};

export const RootKeyInfo = ({
  rootKeyDetails,
}: {
  rootKeyDetails: RootKeySummary;
}) => {
  return (
    <div className="flex gap-5 items-center bg-white dark:bg-black border border-grayA-5 rounded-xl py-5 pl-[18px] pr-[26px]">
      <div className="bg-grayA-5 text-gray-12 size-5 flex items-center justify-center rounded-sm ">
        <Key2 iconSize="sm-regular" />
      </div>
      <div className="flex flex-col gap-1">
        <div className="text-accent-12 text-[13px] font-medium">
          {rootKeyDetails.name ?? "Unnamed Root Key"}
        </div>
        <div className="text-accent-9 text-xs max-w-[160px] truncate">
          {rootKeyDetails.start}...
        </div>
      </div>
    </div>
  );
};
