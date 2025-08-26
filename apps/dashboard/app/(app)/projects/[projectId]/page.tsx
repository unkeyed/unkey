"use client";
import {
  Bolt,
  Book2,
  ChartActivity,
  CircleHalfDottedClock,
  CodeBranch,
  CodeCommit,
  Connections,
  Cube,
  DoubleChevronRight,
  FolderCloud,
  Github,
  Grid,
  Harddrive,
  Heart,
  Location2,
  MessageWriting,
  PaperClip2,
  User,
} from "@unkey/icons";
import { Badge, Button, InfoTooltip, TimestampInfo } from "@unkey/ui";
import { type ReactNode, useCallback, useState } from "react";
import { ProjectNavigation } from "./navigations/project-navigation";
import { ProjectSubNavigation } from "./navigations/project-sub-navigation";

const deploymentDetails: DeploymentDetail[] = [
  {
    icon: <Github className="size-[14px] text-white" />,
    label: "Repository",
    content: (
      <>
        <span className="text-gray-12 font-medium">acme</span>/acme
      </>
    ),
  },
  {
    icon: <CodeBranch className="size-[14px] text-white" size="md-regular" />,
    label: "Branch",
    content: <span className="text-gray-12 font-medium">main</span>,
  },
  {
    icon: <CodeCommit className="size-[14px] text-white" size="md-regular" />,
    label: "Commit",
    content: <span className="text-gray-12 font-medium">e5f6a7b</span>,
  },
  {
    icon: <MessageWriting className="size-[14px] text-white" size="md-regular" />,
    label: "Description",
    content: (
      <div className="truncate max-w-[150px] min-w-0">
        <span className="text-gray-12 font-medium">Add auth routes + logging</span>
      </div>
    ),
  },
  {
    icon: <FolderCloud className="size-[14px] text-white" size="md-regular" />,
    label: "Image", // This was labeled "Description" but should be "Image"
    content: (
      <>
        <span className="text-gray-12 font-medium">unkey</span>:latest
      </>
    ),
  },
  {
    icon: <User className="size-[14px] text-white" size="md-regular" />,
    label: "Author",
    content: (
      <div className="flex gap-2 items-center">
        <img
          src="https://avatars.githubusercontent.com/u/138932600?s=48&v=4"
          alt="Author"
          className="rounded-full size-5"
        />
        <span className="font-medium text-grayA-12">Oz</span>
      </div>
    ),
  },
  {
    icon: <User className="size-[14px] text-white" size="md-regular" />,
    label: "Created",
    content: <TimestampInfo value={Date.now()} className="font-medium text-grayA-12" />,
  },
];

const runtimeSettings: DeploymentDetail[] = [
  {
    icon: <Connections className="text-white" size="md-regular" />,
    label: "Instances",
    content: (
      <div className="text-grayA-10">
        <span className="text-gray-12 font-medium">4</span>vm
      </div>
    ),
  },
  {
    icon: <Location2 className="size-[14px] text-white" size="md-regular" />,
    label: "Regions",
    content: (
      <div className="flex flex-wrap gap-1">
        {["eu-west-2", "us-east-1", "ap-southeast-1"].map((region) => (
          <span
            key={region}
            className="px-1.5 py-1 bg-grayA-3 rounded text-gray-12 text-xs font-mono"
          >
            {region}
          </span>
        ))}
      </div>
    ),
  },
  {
    icon: <Bolt className="size-[14px] text-white" size="md-regular" />,
    label: "CPU",
    content: (
      <div className="text-grayA-10">
        <span className="text-gray-12 font-medium">32</span>vCPUs
      </div>
    ),
  },
  {
    icon: <Grid className="size-[14px] text-white" size="md-regular" />,
    label: "Memory",
    content: (
      <div className="text-grayA-10">
        <span className="text-gray-12 font-medium">512</span>mb
      </div>
    ),
  },
  {
    icon: <Harddrive className="size-[14px] text-white" size="md-regular" />,
    label: "Storage",
    content: (
      <div className="text-grayA-10">
        <span className="text-gray-12 font-medium">1024</span>mb
      </div>
    ),
  },
  {
    icon: <Heart className="size-[14px] text-white" size="md-regular" />,
    label: "Storage",
    content: (
      <div className="flex flex-col justify-center gap-2">
        <div className="gap-2 items-center flex">
          <Badge variant="success" className="font-medium">
            GET
          </Badge>
          <div className="text-grayA-10">
            /<span className="text-gray-12 font-medium">health</span>
          </div>
        </div>
        <div className="flex items-center gap-1 text-grayA-10">
          <div>every</div>
          <div>
            <span className="text-gray-12 font-medium">30</span>s
          </div>
        </div>
      </div>
    ),
  },
  {
    icon: <ChartActivity className="size-[14px] text-white" size="md-regular" />,
    label: "Scaling",
    content: (
      <div className="text-grayA-10">
        <div>
          <span className="text-gray-12 font-medium">0</span> to{" "}
          <span className="text-gray-12 font-medium">5</span> instances
        </div>
        <div className="text-xs mt-0.5">
          at <span className="text-gray-12 font-medium">80%</span> CPU threshold
        </div>
      </div>
    ),
  },
];

const buildInfo: DeploymentDetail[] = [
  {
    icon: <PaperClip2 className="text-white" size="md-regular" />,
    label: "Image size",
    content: (
      <div className="text-grayA-10">
        <span className="text-gray-12 font-medium">210</span>mb
      </div>
    ),
  },
  {
    icon: <CircleHalfDottedClock className="text-white" size="md-regular" />,
    label: "Build time",
    content: (
      <div className="text-grayA-10">
        <span className="text-gray-12 font-medium">45</span>s
      </div>
    ),
  },
];

export default function ProjectDetails({
  params: { projectId },
}: {
  params: { projectId: string };
}) {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  }, []);

  return (
    <div>
      <ProjectNavigation projectId={projectId} />
      <div>
        <ProjectSubNavigation onMount={handleDistanceToTop} />
        <div className="flex">
          <div className="flex-1 bg-success-10">Overview</div>
          <div
            className="fixed right-0 bg-gray-1 border-l border-grayA-4 w-[320px] overflow-y-auto"
            style={{
              top: `${tableDistanceToTop}px`,
              height: `calc(100vh - ${tableDistanceToTop}px)`,
            }}
          >
            {/* Details Section */}
            <div className="h-10 flex items-center justify-between border-b border-grayA-4 px-4">
              <div className="items-center flex gap-2.5 pl-0.5 py-2">
                <Book2 size="md-medium" />
                <span className="text-accent-12 font-medium leading-4 text-[13px]">Details</span>
              </div>
              <Button variant="ghost" size="icon">
                <DoubleChevronRight size="lg-medium" className="text-gray-8" />
              </Button>
            </div>

            {/* Domains Section */}
            <div className="h-20 mt-4 px-4">
              <div className="items-center flex gap-4">
                <Button
                  variant="outline"
                  className="size-12 p-0 [&>svg]:size-[18px] bg-grayA-3 border border-grayA-3 rounded-xl"
                >
                  <Cube size="xl-medium" />
                </Button>
                <div className="flex flex-col gap-1">
                  <span className="text-accent-12 font-medium text-[13px] leading-4">
                    dashboard
                  </span>
                  <div className="gap-2 items-center flex">
                    <span className="text-gray-9 text-[13px] leading-4">api.gateway.com</span>
                    <InfoTooltip
                      position={{
                        side: "bottom",
                      }}
                      content={
                        <div className="space-y-1">
                          {["staging.gateway.com", "dev.gateway.com"].map((region) => (
                            <div
                              key={region}
                              className="text-xs font-medium flex items-center gap-1.5"
                            >
                              <div className="w-1 h-1 bg-gray-8 rounded-full" />
                              {region}
                            </div>
                          ))}
                        </div>
                      }
                    >
                      <div className="rounded-full px-1.5 py-0.5 bg-grayA-3 text-gray-12 text-xs leading-[18px] font-mono tabular-nums">
                        +2
                      </div>
                    </InfoTooltip>
                  </div>
                </div>
              </div>
            </div>

            {/* Active Deployment Section */}
            <div className="px-4">
              <div className="flex items-center gap-3">
                <div className="text-gray-9 text-xs flex-shrink-0">Active deployment</div>
                <div className="h-px bg-grayA-3 w-full" />
              </div>
              <div className="mt-5" />
              <div className="flex flex-col gap-3">
                {deploymentDetails.map((detail, index) => (
                  <DetailRow
                    key={`${detail.label}-${index}`}
                    icon={detail.icon}
                    label={detail.label}
                  >
                    {detail.content}
                  </DetailRow>
                ))}
              </div>
            </div>

            {/* Runtime Section */}
            <div className="px-4 mt-7">
              <div className="flex items-center gap-3">
                <div className="text-gray-9 text-xs flex-shrink-0">Runtime settings</div>
                <div className="h-px bg-grayA-3 w-full" />
              </div>
              <div className="mt-5" />
              <div className="flex flex-col gap-3">
                {runtimeSettings.map((detail, index) => (
                  <DetailRow
                    key={`${detail.label}-${index}`}
                    icon={detail.icon}
                    label={detail.label}
                  >
                    {detail.content}
                  </DetailRow>
                ))}
              </div>
            </div>

            {/* Build Info section*/}
            <div className="px-4 mt-7">
              <div className="flex items-center gap-3">
                <div className="text-gray-9 text-xs flex-shrink-0">Build Info</div>
                <div className="h-px bg-grayA-3 w-full" />
              </div>
              <div className="mt-5" />
              <div className="flex flex-col gap-3">
                {buildInfo.map((detail, index) => (
                  <DetailRow
                    key={`${detail.label}-${index}`}
                    icon={detail.icon}
                    label={detail.label}
                  >
                    {detail.content}
                  </DetailRow>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

type DetailRowProps = {
  icon: ReactNode;
  label: string;
  children: ReactNode;
};

function DetailRow({ icon, label, children }: DetailRowProps) {
  return (
    <div className="flex items-center">
      <div className="flex items-center gap-3 w-[120px]">
        <div className="bg-grayA-3 size-[22px] rounded-md flex items-center justify-center">
          {icon}
        </div>
        <span className="text-grayA-11 text-xs">{label}</span>
      </div>
      <div className="text-grayA-11 text-[13px] min-w-0 flex-1">{children}</div>
    </div>
  );
}

type DeploymentDetail = {
  icon: ReactNode;
  label: string;
  content: ReactNode;
};
