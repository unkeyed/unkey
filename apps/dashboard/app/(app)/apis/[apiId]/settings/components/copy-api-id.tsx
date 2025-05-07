import { toast } from "@/components/ui/toaster";
import { Clone } from "@unkey/icons";
import { Input, SettingCard } from "@unkey/ui";

export const CopyApiId = ({ apiId }: { apiId: string }) => {
  return (
    <SettingCard
      title={"API ID"}
      description={
        <div className="max-w-[380px]">An identifier for the API, used in some API calls.</div>
      }
      border="bottom"
      contentWidth="w-full lg:w-[320px] justify-end items-end"
    >
      <div className="flex flex-row justify-end items-center gap-x-2 mt-1">
        <Input
          className="min-w-[315px] focus:ring-0 focus:ring-offset-0"
          readOnly
          defaultValue={apiId}
          placeholder="API ID"
          rightIcon={
            <button
              type="button"
              onClick={() => {
                navigator.clipboard.writeText(apiId);
                toast.success("Copied to clipboard", {
                  description: apiId,
                });
              }}
            >
              <Clone size="md-regular" />
            </button>
          }
        />
      </div>
    </SettingCard>
  );
};
