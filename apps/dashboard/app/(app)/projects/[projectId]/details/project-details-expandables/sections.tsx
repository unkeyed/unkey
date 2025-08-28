import {
  Bolt,
  ChartActivity,
  CircleHalfDottedClock,
  CodeBranch,
  CodeCommit,
  Connections,
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
import { Badge, TimestampInfo } from "@unkey/ui";
import type { ReactNode } from "react";

export type DetailItem = {
  icon: ReactNode;
  label: string;
  content: ReactNode;
  alignment?: "center" | "start";
};

export type DetailSection = {
  title: string;

  items: DetailItem[];
};

export const createDetailSections = (): DetailSection[] => [
  {
    title: "Active deployment",
    items: [
      {
        icon: <Github className="size-[16px] text-gray-12" />,
        label: "Repository",
        content: (
          <div className="text-grayA-10">
            <span className="text-gray-12 font-medium">acme</span>/acme
          </div>
        ),
      },
      {
        icon: <CodeBranch className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Branch",
        content: <span className="text-gray-12 font-medium">main</span>,
      },
      {
        icon: <CodeCommit className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Commit",
        content: <span className="text-gray-12 font-medium">e5f6a7b</span>,
      },
      {
        icon: <MessageWriting className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Description",
        content: (
          <div className="truncate max-w-[150px] min-w-0">
            <span className="text-gray-12 font-medium">Add auth routes + logging</span>
          </div>
        ),
      },
      {
        icon: <FolderCloud className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Image",
        content: (
          <div className="text-grayA-10">
            <span className="text-gray-12 font-medium">unkey</span>:latest
          </div>
        ),
      },
      {
        icon: <User className="size-[14px] text-gray-12" size="md-regular" />,
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
        icon: <CircleHalfDottedClock className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Created",
        content: (
          <TimestampInfo value={Date.now()} className="font-medium text-grayA-12 text-[13px]" />
        ),
      },
    ],
  },
  {
    title: "Runtime settings",
    items: [
      {
        icon: <Connections className="text-gray-12" size="md-regular" />,
        label: "Instances",
        content: (
          <div className="text-grayA-10">
            <span className="text-gray-12 font-medium">4</span>vm
          </div>
        ),
      },
      {
        icon: <Location2 className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Regions",
        alignment: "start",
        content: (
          <div className="flex flex-wrap gap-1 font-medium">
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
        icon: <Bolt className="size-[14px] text-gray-12" size="md-regular" />,
        label: "CPU",
        content: (
          <div className="text-grayA-10">
            <span className="text-gray-12 font-medium">32</span>vCPUs
          </div>
        ),
      },
      {
        icon: <Grid className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Memory",
        content: (
          <div className="text-grayA-10">
            <span className="text-gray-12 font-medium">512</span>mb
          </div>
        ),
      },
      {
        icon: <Harddrive className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Storage",
        content: (
          <div className="text-grayA-10">
            <span className="text-gray-12 font-medium">1024</span>mb
          </div>
        ),
      },
      {
        icon: <Heart className="size-[14px] text-gray-12" size="md-regular" />,
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
        icon: <ChartActivity className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Scaling",
        alignment: "start",
        content: (
          <div className="text-grayA-10">
            <div>
              <span className="text-gray-12 font-medium">0</span> to{" "}
              <span className="text-gray-12 font-medium">5</span> instances
            </div>
            <div className="mt-0.5">
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
        icon: <PaperClip2 className="text-gray-12" size="md-regular" />,
        label: "Image size",
        content: (
          <div className="text-grayA-10">
            <span className="text-gray-12 font-medium">210</span>mb
          </div>
        ),
      },
      {
        icon: <CircleHalfDottedClock className="text-gray-12" size="md-regular" />,
        label: "Build time",
        content: (
          <div className="text-grayA-10">
            <span className="text-gray-12 font-medium">45</span>s
          </div>
        ),
      },
      {
        icon: <Bolt className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Build status",
        content: (
          <Badge variant="success" className="font-medium">
            Success
          </Badge>
        ),
      },
      {
        icon: <Gear className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Base image",
        content: (
          <div className="text-grayA-10">
            <span className="text-gray-12 font-medium">node</span>:18-alpine
          </div>
        ),
      },
      {
        icon: <CircleHalfDottedClock className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Built At",
        content: (
          <TimestampInfo
            value={Date.now() - 300000}
            className="font-medium text-grayA-12 text-[13px]"
          />
        ),
      },
    ],
  },
];
