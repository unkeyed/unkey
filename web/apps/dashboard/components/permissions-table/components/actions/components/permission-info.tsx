import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { IconPage2Outline12 } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";

export const PermissionInfo = ({
  permissionDetails,
}: {
  permissionDetails: Permission;
}) => {
  return (
    <div className="flex gap-5 items-center bg-raised border rounded-xl py-5 pl-4.5 pr-6.5">
      <div className="bg-grayA-5 text-gray-12 size-5 flex items-center justify-center rounded-sm">
        <IconPage2Outline12 />
      </div>
      <div className="flex flex-col gap-1">
        <div className="text-gray-12 text-[13px] font-medium">{permissionDetails.name}</div>
        {permissionDetails.description && (
          <InfoTooltip
            variant="inverted"
            content={permissionDetails.description}
            position={{ side: "bottom", align: "center" }}
            asChild
          >
            <div className="text-gray-9 text-xs max-w-40 truncate">
              {permissionDetails.description}
            </div>
          </InfoTooltip>
        )}
      </div>
    </div>
  );
};
