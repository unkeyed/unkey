import { IconKey2Outline12 } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";

export const KeyInfo = ({
  keyDetails,
}: {
  keyDetails: { name?: string | null; id: string };
}) => {
  return (
    <div className="flex gap-5 items-center bg-raised border rounded-xl py-5 pl-[18px] pr-[26px]">
      <div className="bg-grayA-5 text-gray-12 size-5 shrink-0 flex items-center justify-center rounded-sm ">
        <IconKey2Outline12 />
      </div>
      <div className="flex flex-col gap-1 min-w-0">
        <div className="text-gray-12 text-xs font-mono truncate">{keyDetails.id}</div>
        <InfoTooltip
          content={keyDetails.name}
          position={{ side: "bottom", align: "center" }}
          asChild
          disabled={!keyDetails.name}
        >
          <div className="text-gray-9 text-xs truncate">{keyDetails.name ?? "Unnamed Key"}</div>
        </InfoTooltip>
      </div>
    </div>
  );
};
