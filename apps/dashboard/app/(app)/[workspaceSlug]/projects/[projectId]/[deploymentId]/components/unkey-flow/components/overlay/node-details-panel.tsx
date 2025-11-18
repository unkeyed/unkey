import { Book2, ChevronExpandY, DoubleChevronRight } from "@unkey/icons";
import {
  InfoTooltip,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { CardHeader } from "../nodes/deploy-node";
import { type DeploymentNode, REGION_INFO } from "../nodes/types";

type NodeDetailsPanelProps = {
  node?: DeploymentNode;
};

export const NodeDetailsPanel = ({ node }: NodeDetailsPanelProps) => {
  if (!node) {
    return null;
  }
  const { flagCode, rps, cpu, memory, health, zones } = node.metadata;
  const regionInfo = REGION_INFO[flagCode];
  return (
    <div
      className={cn(
        "absolute top-14 right-4 rounded-xl bg-white dark:bg-black border border-grayA-4 shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)] overflow-y-auto pb-4 pointer-events-auto min-w-[360px]",
        "transition-all duration-300 ease-out",
        node ? "opacity-100 translate-y-0" : "opacity-0 -translate-y-2 pointer-events-none",
      )}
    >
      <div className="flex flex-col items-center">
        {/*Header*/}
        <div className="flex  items-center justify-between h-12 border-b border-grayA-4 w-full px-3 py-2.5 ">
          <div className="flex gap-2.5 items-center p-2 border rounded-lg border-grayA-5 bg-grayA-2 h-[26px]">
            <Book2 className="text-gray-12" iconSize="sm-regular" />
            <span className="text-accent-12 font-medium text-[13px] leading-4">Details</span>
          </div>
          <DoubleChevronRight className="text-gray-8 shrink-0" iconSize="lg-regular" />
        </div>
        {/*Node Info*/}

        <div className="flex items-center justify-between w-full px-3 py-4">
          <CardHeader
            variant="panel"
            icon={
              <InfoTooltip
                content={`AWS region ${node.label} (${regionInfo.location})`}
                variant="primary"
                className="px-2.5 py-1 rounded-[10px] bg-white dark:bg-blackA-12 text-xs z-30"
                position={{ align: "center", side: "top", sideOffset: 5 }}
              >
                <div className="border rounded-[10px] border-grayA-3 size-12 bg-grayA-3 flex items-center justify-center">
                  <img
                    src={`/images/flags/${flagCode}.svg`}
                    alt={flagCode}
                    className="size-[22px]"
                  />
                </div>
              </InfoTooltip>
            }
            title={node.label}
            subtitle={`${zones} availability ${zones === 1 ? "zone" : "zones"}`}
            health={health}
          />
        </div>

        <div className="flex px-4 w-full">
          <div className="flex items-center gap-3 w-full">
            <div className="text-gray-9 text-xs whitespace-nowrap">Runtime metrics</div>
            <div className="h-0.5 bg-grayA-3 rounded-sm flex-1 min-w-[115px]" />
            <div className="flex items-center gap-2 shrink-0">
              <Select>
                <SelectTrigger
                  className="rounded-lg !px-2 !py-1.5 text-gray-10 text-xs !min-h-[26px]"
                  rightIcon={<ChevronExpandY className="ml-2 text-gray-10" />}
                >
                  <SelectValue placeholder="24H" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="option1">Option 1</SelectItem>
                  <SelectItem value="option2">Option 2</SelectItem>
                  <SelectItem value="option3">Option 3</SelectItem>
                </SelectContent>
              </Select>
              <Select>
                <SelectTrigger
                  className="rounded-lg !px-2 !py-1.5 text-gray-10 text-xs !min-h-[26px]"
                  rightIcon={<ChevronExpandY className="ml-2 text-gray-10" />}
                >
                  <SelectValue placeholder="PST" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="option1">Option 1</SelectItem>
                  <SelectItem value="option2">Option 2</SelectItem>
                  <SelectItem value="option3">Option 3</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
