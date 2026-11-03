import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { IconKey2Outline12 } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";

export const RoleInfo = ({ roleDetails }: { roleDetails: RoleBasic }) => {
  return (
    <div className="flex gap-5 items-center bg-raised border rounded-xl py-5 pl-[18px] pr-[26px]">
      <div className="bg-grayA-5 text-gray-12 size-5 shrink-0 flex items-center justify-center rounded-sm ">
        <IconKey2Outline12 />
      </div>
      <div className="flex flex-col gap-1 min-w-0">
        <div className="text-gray-12 text-sm font-medium truncate">
          {roleDetails.name ?? "Unnamed Role"}
        </div>
        <InfoTooltip
          content={roleDetails.description}
          position={{ side: "bottom", align: "center" }}
          asChild
          disabled={!roleDetails.description}
        >
          <div className="text-gray-9 text-xs truncate">{roleDetails.description}</div>
        </InfoTooltip>
      </div>
    </div>
  );
};
