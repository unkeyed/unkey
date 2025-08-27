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
  Gear,
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

type DetailItem = {
  icon: ReactNode;
  label: string;
  content: ReactNode;
  alignment?: "center" | "start";
};

type DetailSection = {
  title: string;
  items: DetailItem[];
};

const createDetailSections = (): DetailSection[] => [
  {
    title: "Active deployment",
    items: [
      {
        icon: <Github className="size-[14px] text-white" />,
        label: "Repository",
        content: (
          <div className="text-grayA-10">
            <span className="text-gray-12 font-medium">acme</span>/acme
          </div>
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
        label: "Image",
        content: (
          <div className="text-grayA-10">
            <span className="text-gray-12 font-medium">unkey</span>:latest
          </div>
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
        content: <TimestampInfo value={Date.now()} className="font-medium text-grayA-12 text-sm" />,
      },
    ],
  },
  {
    title: "Runtime settings",
    items: [
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
        alignment: "start",
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
        label: "Healthcheck",
        alignment: "start",
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
        alignment: "start",
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
    ],
  },
  {
    title: "Build Info",
    items: [
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
      {
        icon: <Bolt className="size-[14px] text-white" size="md-regular" />,
        label: "Build status",
        content: (
          <Badge variant="success" className="text-[13px]">
            Success
          </Badge>
        ),
      },
      {
        icon: <Gear className="size-[14px] text-white" size="md-regular" />,
        label: "Base image",
        content: (
          <div className="text-grayA-10">
            <span className="text-gray-12 font-medium">node</span>:18-alpine
          </div>
        ),
      },
      {
        icon: <CircleHalfDottedClock className="size-[14px] text-white" size="md-regular" />,
        label: "Built At",
        content: (
          <TimestampInfo
            value={Date.now() - 300000}
            className="font-medium text-grayA-12 text-sm"
          />
        ),
      },
    ],
  },
];

type DetailRowProps = {
  icon: ReactNode;
  label: string;
  children: ReactNode;
  alignment?: "center" | "start";
};

function DetailRow({ icon, label, children, alignment = "center" }: DetailRowProps) {
  const alignmentClass = alignment === "start" ? "items-start" : "items-center";

  return (
    <div className={`flex ${alignmentClass}`}>
      <div className="flex items-center gap-3 w-[135px]">
        <div className="bg-grayA-3 size-[22px] rounded-md flex items-center justify-center">
          {icon}
        </div>
        <span className="text-grayA-11 text-[13px]">{label}</span>
      </div>
      <div className="text-grayA-11 text-sm min-w-0 flex-1">{children}</div>
    </div>
  );
}

type DetailSectionProps = {
  title: string;
  items: DetailItem[];
  isFirst?: boolean;
};

function DetailSection({ title, items, isFirst = false }: DetailSectionProps) {
  return (
    <div className={`px-4 ${isFirst ? "" : "mt-7"}`}>
      <div className="flex items-center gap-3">
        <div className="text-gray-9 text-sm flex-shrink-0">{title}</div>
        <div className="h-px bg-grayA-3 w-full" />
      </div>
      <div className="mt-5" />
      <div className="flex flex-col gap-3.5">
        {items.map((item, index) => (
          <DetailRow
            key={`${item.label}-${index}`}
            icon={item.icon}
            label={item.label}
            alignment={item.alignment}
          >
            {item.content}
          </DetailRow>
        ))}
      </div>
    </div>
  );
}

export default function ProjectDetails({
  params: { projectId },
}: {
  params: { projectId: string };
}) {
  const [tableDistanceToTop, setTableDistanceToTop] = useState(0);

  const handleDistanceToTop = useCallback((distanceToTop: number) => {
    setTableDistanceToTop(distanceToTop);
  }, []);

  const detailSections = createDetailSections();

  return (
    <div>
      <ProjectNavigation projectId={projectId} />
      <div>
        <ProjectSubNavigation onMount={handleDistanceToTop} />
        <div className="flex">
          <div className="flex-1 bg-success-10">Overview</div>
          <div
            className="fixed right-0 bg-gray-1 border-l border-grayA-4 w-[360px] overflow-y-auto"
            style={{
              top: `${tableDistanceToTop}px`,
              height: `calc(100vh - ${tableDistanceToTop}px)`,
            }}
          >
            {/* Details Header */}
            <div className="h-10 flex items-center justify-between border-b border-grayA-4 px-4">
              <div className="items-center flex gap-2.5 pl-0.5 py-2">
                <Book2 size="md-medium" />
                <span className="text-accent-12 font-medium text-sm">Details</span>
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
                  className="size-12 p-0 bg-grayA-3 border border-grayA-3 rounded-xl"
                >
                  <Cube size="2xl-medium" className="!size-[20px]" />
                </Button>
                <div className="flex flex-col gap-1">
                  <span className="text-accent-12 font-medium text-sm">dashboard</span>
                  <div className="gap-2 items-center flex">
                    <span className="text-gray-9 text-sm">api.gateway.com</span>
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

            {/* Dynamic Detail Sections */}
            {detailSections.map((section, index) => (
              <DetailSection
                key={section.title}
                title={section.title}
                items={section.items}
                isFirst={index === 0}
              />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
