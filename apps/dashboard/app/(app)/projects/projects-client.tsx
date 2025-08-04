"use client";

import { CodeBranch, Cube, Dots, User } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { LogsProvider } from "../logs/context/logs";
import { LogsControlCloud } from "./_components/control-cloud";
import { LogsControls } from "./_components/controls";
import { ProjectsNavigation } from "./_components/navigation";

export function ProjectsClient() {
  return (
    <div>
      <ProjectsNavigation />
      <LogsProvider>
        <LogsControls />
        <LogsControlCloud />
      </LogsProvider>
      {/*Container*/}
      <div className="p-4">
        {/*Card*/}
        <div className="p-5 flex flex-col border border-grayA-4 rounded-2xl w-[380px] h-[190px] gap-5">
          {/*Top Section*/}
          <div className="flex gap-4 items-center">
            <div className="size-10 bg-grayA-3 border border-grayA-3 rounded-[10px] flex items-center justify-center shrink-0">
              <Cube size="xl-medium" className="text-gray-12 shrink-0" />
            </div>
            <div className="flex flex-col w-full gap-[10px] py-[5px]">
              {/*Top Section > Project Name*/}
              <div className="font-medium text-sm leading-[10px] text-accent-12">dashboard</div>
              {/*Top Section > Domains/Hostnames*/}
              <div className="font-medium text-xs leading-[10px] text-gray-11">api.gateway.com</div>
            </div>
            {/*Top Section > Project actions*/}
            <Button variant="ghost" size="icon" className="mb-auto" title="Project actions">
              <Dots size="sm-regular" />
            </Button>
          </div>
          <div className="flex flex-col gap-2">
            {/*Middle Section > Last commit title*/}
            <div className="text-[13px] font-medium text-accent-12 leading-5">
              feat: add paginated tRPC endpoint for projects (#3697)
            </div>
            <div className="flex gap-2 items-center">
              <span className="text-xs text-gray-11">Jul 24 on</span>
              <CodeBranch className="text-gray-12" size="sm-regular" />
              <span className="text-xs text-gray-12">main</span>
              <span className="text-xs text-gray-9">by</span>
              <div className="border border-grayA-6 items-center justify-center rounded-full size-[18px] flex">
                <User className="text-gray-11" size="sm-regular" />
              </div>
              <span className="text-xs text-gray-12 font-medium">Oz</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
