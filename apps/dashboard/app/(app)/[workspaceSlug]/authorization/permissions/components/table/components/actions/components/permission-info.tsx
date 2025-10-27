import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { Page2 } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";

export const PermissionInfo = ({
  permissionDetails,
}: {
  permissionDetails: Permission;
}) => {
  return (
    <div className="flex gap-5 items-center bg-white dark:bg-black border border-grayA-5 rounded-xl py-5 pl-[18px] pr-[26px]">
      <div className="bg-grayA-5 text-gray-12 size-5 flex items-center justify-center rounded ">
        <Page2 iconSize="sm-regular" />
      </div>
      <div className="flex flex-col gap-1">
        <div className="text-accent-12 text-[13px] font-medium">{permissionDetails.name}</div>
        <InfoTooltip
          variant="inverted"
          content={permissionDetails.name}
          position={{ side: "bottom", align: "center" }}
          asChild
          disabled={!permissionDetails.name}
        >
          <div className="text-accent-9 text-xs max-w-[160px] truncate">
            {permissionDetails.description}
          </div>
        </InfoTooltip>
      </div>
    </div>
  );
};
