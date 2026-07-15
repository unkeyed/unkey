import { CopyButton, InfoTooltip } from "@unkey/ui";

type OverrideIdentifierCellProps = {
  identifier: string;
};

export const OverrideIdentifierCell = ({ identifier }: OverrideIdentifierCellProps) => {
  return (
    <div className="inline-flex items-start pl-2">
      <InfoTooltip
        content={
          <div className="flex gap-3">
            <div className="flex justify-start items-center break-all max-w-[400px] secret">
              {identifier}
            </div>
            <div className="flex flex-col justify-center items-center w-4 secret">
              <CopyButton value={identifier} />
            </div>
          </div>
        }
        position={{ side: "bottom", align: "start" }}
      >
        <pre className="text-[11px] text-gray-11 sm:max-w-[100px] md:max-w-[100px] lg:max-w-[320px] xl:max-w-[600px] truncate secret">
          {identifier}
        </pre>
      </InfoTooltip>
    </div>
  );
};
