"use client";
import { ChevronDown, CircleCheck, Cloud, CodeBranch, CodeCommit, FolderCloud } from "@unkey/icons";
import { Badge, Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { ProjectLayout } from "./project-layout";

export default function ProjectDetails({
  params: { projectId },
}: {
  params: { projectId: string };
}) {
  return (
    <ProjectLayout projectId={projectId}>
      {({ isDetailsOpen }) => (
        <div
          className={cn(
            "flex justify-center transition-all duration-300 ease-in-out",
            isDetailsOpen ? "w-[calc(100vw-616px)]" : "w-[calc(100vw-256px)]",
          )}
        >
          <div className="max-w-[960px] flex flex-col w-full mt-4">
            {/*Active deployment section*/}
            <div className="flex flex-col gap-1">
              <div className="flex items-center gap-2.5 py-1.5 px-2">
                <Cloud size="sm-regular" className="text-gray-9" />
                <div className="text-accent-12 font-medium text-[13px] leading-4">
                  Active Deployment
                </div>
              </div>
              <div className="border border-gray-4 rounded-[14px] w-full pt-[14px] flex justify-between flex-col overflow-hidden">
                <div className="flex w-full justify-between items-center px-[22px] ">
                  <div className="flex gap-5 items-center">
                    <div className="relative">
                      <div
                        className={cn(
                          "size-5 rounded flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100",
                          "bg-grayA-3",
                        )}
                      >
                        <Cloud size="sm-regular" className="text-gray-12" />
                      </div>
                      <div className="absolute -top-0.5 -right-0.5">
                        {/* Radio wave rings */}
                        <div
                          className="absolute inset-0 size-2 bg-successA-9 rounded-full opacity-75"
                          style={{
                            animation: "ping 2s cubic-bezier(0, 0, 0.2, 1) infinite",
                          }}
                        />
                        <div
                          className="absolute inset-0 size-2 bg-successA-10 rounded-full opacity-60"
                          style={{
                            animation: "ping 2s cubic-bezier(0, 0, 0.2, 1) infinite",
                            animationDelay: "0.15s",
                          }}
                        />
                        <div
                          className="absolute inset-0 size-2 bg-successA-11 rounded-full opacity-40"
                          style={{
                            animation: "ping 2s cubic-bezier(0, 0, 0.2, 1) infinite",
                            animationDelay: "0.3s",
                          }}
                        />
                        <div
                          className="absolute inset-0 size-2 bg-successA-12 rounded-full opacity-25"
                          style={{
                            animation: "ping 2s cubic-bezier(0, 0, 0.2, 1) infinite",
                            animationDelay: "0.45s",
                          }}
                        />
                        {/* Center dot */}
                        <div className="relative size-2 bg-successA-9 rounded-full" />
                      </div>
                    </div>
                    <div className="flex flex-col gap-1">
                      <div className="text-accent-12 font-medium text-xs">v_alpha001</div>
                      <div className="text-gray-9 text-xs">Add auth routes + logging</div>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <Badge variant="success" className="text-successA-11 font-medium">
                      <div className="flex items-center gap-2">
                        <CircleCheck />
                        Active
                      </div>
                    </Badge>
                    <div className="items-center flex gap-2">
                      <div className="flex gap-2 items-center">
                        <span className="text-gray-9 text-xs"> Created by</span>
                        <img
                          src="https://avatars.githubusercontent.com/u/138932600?s=48&v=4"
                          alt="Author"
                          className="rounded-full size-5"
                        />
                        <span className="font-medium text-grayA-12 text-xs">Oz</span>
                      </div>
                    </div>
                  </div>
                </div>
                <div className="bg-gray-1 rounded-b-[14px]">
                  <div className="relative h-4 flex items-center justify-center">
                    <div className="absolute top-0 left-0 right-0 h-4 border-b rounded-b-[14px] bg-white dark:bg-black" />
                  </div>
                  <div className="pb-2.5 pt-2 flex justify-between items-center px-3">
                    <div className="flex items-center gap-2.5">
                      <span className="text-grayA-9 text-xs">a day ago</span>
                      <div className="flex items-center gap-1.5">
                        <div className="gap-2 flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100 bg-grayA-3 p-1.5 h-[22px] rounded-md">
                          <CodeBranch size="md-medium" className="text-gray-12" />
                          <span className="text-grayA-9 text-xs">main</span>
                        </div>
                        <div className="gap-2 flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100 bg-grayA-3 p-1.5 h-[22px] rounded-md">
                          <CodeCommit size="md-medium" className="text-gray-12" />
                          <span className="text-grayA-9 text-xs">e5f6a7b</span>
                        </div>
                      </div>

                      <span className="text-grayA-9 text-xs">using image</span>
                      <div className="gap-2 flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100 bg-grayA-3 p-1.5 h-[22px] rounded-md">
                        <FolderCloud size="md-medium" className="text-gray-12" />
                        <div className="text-grayA-10 text-xs">
                          <span className="text-gray-12 font-medium ">unkey</span>
                          :latest
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-1.5">
                      <div className="text-grayA-9 text-xs">Build logs</div>
                      <Button size="icon" variant="ghost">
                        <ChevronDown className="text-grayA-9 !size-3" />
                      </Button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </ProjectLayout>
  );
}
