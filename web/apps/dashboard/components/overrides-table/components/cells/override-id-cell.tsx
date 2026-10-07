import { CopyButton, InfoHoverCard } from "@unkey/ui";

type OverrideIdCellProps = {
  id: string;
};

export const OverrideIdCell = ({ id }: OverrideIdCellProps) => {
  return (
    <div className="pl-2">
      <InfoHoverCard
        content={
          <div className="inline-flex justify-center gap-3 items-center font-mono text-xs text-gray-11">
            <span className="secret">{id}</span>
            <CopyButton className="secret" value={id} />
          </div>
        }
        position={{ side: "bottom", align: "start" }}
      >
        <div className="font-mono text-xs text-gray-11 sm:max-w-[100px] md:max-w-[100px] lg:max-w-full truncate">
          {id}
        </div>
      </InfoHoverCard>
    </div>
  );
};
