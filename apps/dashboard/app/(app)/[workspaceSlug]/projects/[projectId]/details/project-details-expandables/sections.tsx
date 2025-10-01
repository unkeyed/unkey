import type { Deployment } from "@/lib/collections";
import {
  Bolt,
  ChartActivity,
  CircleHalfDottedClock,
  CodeBranch,
  CodeCommit,
  Connections,
  Github,
  Grid,
  Harddrive,
  Heart,
  Location2,
  MessageWriting,
  User,
} from "@unkey/icons";
import { Badge, TimestampInfo } from "@unkey/ui";
import type { ReactNode } from "react";
import { RepoDisplay } from "../../../_components/list/repo-display";
import { Avatar } from "../active-deployment-card/git-avatar";

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

export const createDetailSections = (
  details: Deployment & { repository: string | null },
): DetailSection[] => [
  {
    title: "Active deployment",
    items: [
      {
        icon: <Github className="size-[16px] text-gray-12" />,
        label: "Repository",
        content: (
          <RepoDisplay
            url={details.repository || "—"}
            showIcon={false}
            className="text-gray-12 font-medium"
          />
        ),
      },
      {
        icon: <CodeBranch className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Branch",
        content: (
          <span className="text-gray-12 font-medium truncate max-w-32">{details.gitBranch}</span>
        ),
      },
      {
        icon: <CodeCommit className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Commit",
        content: (
          <span className="text-gray-12 font-medium">
            {(details.gitCommitSha ?? "").slice(0, 7)}
          </span>
        ),
      },
      {
        icon: <MessageWriting className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Description",
        content: (
          <div className="truncate max-w-[150px] min-w-0">
            <span className="text-gray-12 font-medium">{details.gitCommitMessage}</span>
          </div>
        ),
      },
      {
        icon: <User className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Author",
        content: (
          <div className="flex gap-2 items-center">
            <Avatar
              src={details.gitCommitAuthorAvatarUrl}
              alt={details.gitCommitAuthorHandle ?? ""}
            />
            <span className="font-medium text-grayA-12">{details.gitCommitAuthorHandle}</span>
          </div>
        ),
      },
      {
        icon: <CircleHalfDottedClock className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Created",
        content: (
          <TimestampInfo
            value={details.createdAt}
            className="font-medium text-grayA-12 text-[13px]"
          />
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
            <span className="text-gray-12 font-medium">
              {details.runtimeConfig.regions.reduce((acc, region) => acc + region.vmCount, 0)}
            </span>
            vm
          </div>
        ),
      },
      {
        icon: <Location2 className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Regions",
        alignment: "start",
        content: (
          <div className="flex flex-wrap gap-1 font-medium">
            {details.runtimeConfig.regions.map((region) => (
              <span
                key={region.region}
                className="px-1.5 py-1 bg-grayA-3 rounded text-gray-12 text-xs font-mono"
              >
                {region.region}
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
            <span className="text-gray-12 font-medium">{details.runtimeConfig.cpus}</span>
            vCPUs
          </div>
        ),
      },
      {
        icon: <Grid className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Memory",
        content: (
          <div className="text-grayA-10">
            <span className="text-gray-12 font-medium">{details.runtimeConfig.memory}</span>
            mb
          </div>
        ),
      },
      {
        icon: <Harddrive className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Storage",
        content: (
          <div className="text-grayA-10">
            <span className="text-gray-12 font-medium">20GB</span>
            mb
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
                TODO
              </Badge>
              <div className="text-grayA-10">
                /<span className="text-gray-12 font-medium">TODO</span>
              </div>
            </div>
            <div className="flex items-center gap-1 text-grayA-10">
              <div>every</div>
              <div>
                <span className="text-gray-12 font-medium">TODO</span>s
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
              <span className="text-gray-12 font-medium">{3}</span> to{" "}
              <span className="text-gray-12 font-medium">{6}</span> instances
            </div>
            <div className="mt-0.5">
              at <span className="text-gray-12 font-medium">70%</span> CPU threshold
            </div>
          </div>
        ),
      },
    ],
  },
  /*
  {
    title: "Build Info",
    items: [
      {
        icon: <PaperClip2 className="text-gray-12" size="md-regular" />,
        label: "Image size",
        content: (
          <div className="text-grayA-10">
            <span className="text-gray-12 font-medium">{details.imageSize}</span>
            mb
          </div>
        ),
      },
      {
        icon: <CircleHalfDottedClock className="text-gray-12" size="md-regular" />,
        label: "Build time",
        content: (
          <div className="text-grayA-10">
            <span className="text-gray-12 font-medium">{details.buildTime}</span>s
          </div>
        ),
      },
      {
        icon: <Bolt className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Build status",
        content: (
          <Badge
            variant={
              details.buildStatus === "success"
                ? "success"
                : details.buildStatus === "failed"
                  ? "error"
                  : "secondary"
            }
            className="font-medium"
          >
            {details.buildStatus === "success"
              ? "Success"
              : details.buildStatus === "failed"
                ? "Failed"
                : "Pending"}
          </Badge>
        ),
      },
      {
        icon: <Gear className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Base image",
        content: (
          <div className="text-grayA-10">
            <span className="text-gray-12 font-medium">{details.baseImage}</span>
          </div>
        ),
      },
      {
        icon: <CircleHalfDottedClock className="size-[14px] text-gray-12" size="md-regular" />,
        label: "Built At",
        content: (
          <TimestampInfo
            value={details.builtAt}
            className="font-medium text-grayA-12 text-[13px]"
          />
        ),
      },
    ],
  },
  */
];
