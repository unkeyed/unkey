import { SettingCard } from "@unkey/ui";
import { CopyButton } from "@unkey/ui";

export const CopyKeySpaceId = ({ keySpaceId }: { keySpaceId: string }) => {
  return (
    <SettingCard
      title={"KeySpace ID"}
      description={
        <div className="max-w-[380px]">Identifier for the underlying keyspace.</div>
      }
      border="bottom"
      contentWidth="w-full lg:w-[420px] justify-end"
    >
      {/* TODO: make this a Code component in UI for CopyKeys with optional hidden button like in Code.*/}
      <div className="flex flex-row justify-end items-center">
        <div
          className={
            "flex flex-row justify-between min-w-[327px] pl-4 pr-2 py-1.5 bg-gray-2 dark:bg-black border rounded-lg border-grayA-5"
          }
        >
          <div className="text-sm text-gray-11">{keySpaceId}</div>
          <CopyButton value={keySpaceId} variant="ghost" toastMessage={keySpaceId} />
        </div>
      </div>
    </SettingCard>
  );
};
