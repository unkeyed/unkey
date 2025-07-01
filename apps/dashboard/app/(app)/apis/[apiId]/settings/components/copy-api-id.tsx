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
      contentWidth="w-full lg:w-[420px] justify-end items-center"
    >
      <div className="flex flex-row justify-end items-center gap-x-2">
        <Input
          className="min-w-[322px] focus:ring-0 focus:ring-offset-0 h-9"
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
