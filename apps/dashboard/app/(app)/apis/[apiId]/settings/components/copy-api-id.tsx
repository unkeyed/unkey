"use client";
import { Clone } from "@unkey/icons";
import { SettingCard, toast } from "@unkey/ui";

export const CopyApiId = ({ apiId }: { apiId: string }) => {
  return (
    <SettingCard
      title={"API ID"}
      description={
        <div className="max-w-[380px]">An identifier for the API, used in some API calls.</div>
      }
      border="bottom"
      contentWidth="w-full lg:w-[420px] justify-end items-center"
    >
      <div className="flex flex-row justify-end items-center">
        <div
          className="justify-between flex items-center min-w-[327px] focus:ring-0 focus:ring-offset-0 h-9 w-full pl-4 pr-3 py-2 bg-white dark:bg-black border rounded-lg border-grayA-5"
        >
          {apiId}
          <button
            type="button"
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              navigator.clipboard.writeText(apiId);
              toast.success("Copied to clipboard", {
                description: apiId,
              });
            }}
          >
            <Clone size="lg-regular" />
          </button>
        </div>
      </div>
    </SettingCard>
  );
};
