import { ChevronDown, CodeBranch, CodeCommit, FolderCloud } from "@unkey/icons";
import { Badge, Button, Card } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { STATUS_CONFIG } from ".";
import { StatusIndicator } from "./status-indicator";

export const ActiveDeploymentCardSkeleton = () => (
  <Card className="rounded-[14px] pt-[14px] flex justify-between flex-col overflow-hidden border-gray-4">
    <div className="flex w-full justify-between items-center px-[22px] h-9">
      <div className="flex gap-5 items-center">
        <StatusIndicator />
        <div className="flex flex-col gap-1">
          <div className="h-2.5 w-16 bg-grayA-3 rounded animate-pulse" />
          <div className="h-2.5 w-32 bg-grayA-3 rounded animate-pulse" />
        </div>
      </div>
      <div className="flex items-center gap-4">
        <Badge variant="success" className="text-successA-11 font-medium">
          <div className="flex items-center gap-2">
            {<STATUS_CONFIG.success.icon />}
            {STATUS_CONFIG.success.text}
          </div>
        </Badge>
        <div className="items-center flex gap-2">
          <div className="flex gap-2 items-center">
            <span className="text-gray-9 text-xs">Created by</span>
            <div className="rounded-full size-5 bg-grayA-3 animate-pulse" />
            <div className="h-2.5 w-20 bg-grayA-3 rounded animate-pulse" />
          </div>
        </div>
      </div>
    </div>

    <div className="bg-gray-1 rounded-b-[14px]">
      <div className="relative h-4 flex items-center justify-center">
        <div className="absolute top-0 left-0 right-0 h-4 border-b border-gray-4 rounded-b-[14px] bg-white dark:bg-black" />
      </div>

      <div className="pb-2.5 pt-2 flex justify-between items-center px-3">
        <div className="flex items-center gap-2.5">
          <div className="h-2.5 w-16 bg-grayA-3 rounded animate-pulse" />
          <div className="flex items-center gap-1.5">
            <div className="bg-grayA-3 items-center flex gap-1.5 p-1.5 rounded-md w-fit animate-pulse h-[22px]">
              <CodeBranch size="sm-regular" className="text-gray-12 opacity-50" />
              <div className="h-2 w-12 bg-grayA-4 rounded" />
            </div>
            <div className="bg-grayA-3 items-center flex gap-1.5 p-1.5 rounded-md w-fit animate-pulse h-[22px]">
              <CodeCommit size="sm-regular" className="text-gray-12 opacity-50" />
              <div className="h-2 w-16 bg-grayA-4 rounded" />
            </div>
          </div>
          <div className="text-grayA-9 text-xs">using image</div>
          <div className="bg-grayA-3 items-center flex gap-1.5 p-1.5 rounded-md w-fit animate-pulse h-[22px]">
            <FolderCloud size="sm-regular" className="text-gray-12 opacity-50" />
            <div className="h-2 w-24 bg-grayA-4 rounded" />
          </div>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="text-grayA-9 text-xs">Build logs</div>
          <Button size="icon" variant="ghost" disabled>
            <ChevronDown className={cn("text-grayA-9 !size-3 transition-transform duration-200")} />
          </Button>
        </div>
      </div>
    </div>
  </Card>
);
