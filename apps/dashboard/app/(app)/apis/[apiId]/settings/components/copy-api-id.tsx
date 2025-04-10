import { SettingCard } from "@/components/settings-card";
import { toast } from "@/components/ui/toaster";
import { Clone } from "@unkey/icons";
import { Input } from "@unkey/ui";

export const CopyApiId = ({ apiId }: { apiId: string }) => {
  return (
    <SettingCard
      className="pt-8 pb-8 mx-0"
      title={
        <div className="flex items-center justify-start gap-2.5">
          <span className="text-sm font-medium text-accent-12">API ID</span>
        </div>
      }
      description={
        <div className="font-normal text-[13px] max-w-[380px]">
          An identifier for the API, used in some API calls.
        </div>
      }
      border="bottom"
    >
      <Input
        className="w-[320px] focus:ring-0 focus:ring-offset-0"
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
            <Clone size="md-regular" className="text-gray-12" />
          </button>
        }
      />
    </SettingCard>
  );
};
