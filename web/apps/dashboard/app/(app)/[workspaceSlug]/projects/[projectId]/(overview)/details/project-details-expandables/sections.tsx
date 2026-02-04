import type { Deployment } from "@/lib/collections";
import { formatCpu, formatMemory } from "@/lib/utils/deployment-formatters";
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
import { TimestampInfo } from "@unkey/ui";
import type { ReactNode } from "react";
import { RepoDisplay } from "../../../../_components/list/repo-display";
import { Avatar } from "../../../components/git-avatar";
import { OpenApiDiff } from "./sections/open-api-diff";

export type DetailItem = {
  icon: ReactNode | null;
  label: string | null;
  disabled?: boolean;
  content: ReactNode;
  alignment?: "center" | "start";
};

export type DetailSection = {
  title: string;
  disabled?: boolean;
  items: DetailItem[];
};

export const createDetailSections = (
  details: Deployment & { repository: string | null },
): DetailSection[] => {
  return [
    {
      title: "OpenAPI changes",
      disabled: true,
      items: [
        {
          icon: null,
          label: null,
          alignment: "start",
          content: <OpenApiDiff />,
        },
      ],
    },
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
          icon: <CodeBranch className="size-[14px] text-gray-12" iconSize="md-regular" />,
          label: "Branch",
          content: (
            <span className="text-gray-12 font-medium truncate max-w-32">{details.gitBranch}</span>
          ),
        },
        {
          icon: <CodeCommit className="size-[14px] text-gray-12" iconSize="md-regular" />,
          label: "Commit",
          content: (
            <span className="text-gray-12 font-medium">
              {(details.gitCommitSha ?? "").slice(0, 7)}
            </span>
          ),
        },
        {
          icon: <MessageWriting className="size-[14px] text-gray-12" iconSize="md-regular" />,
          label: "Description",
          content: (
            <div className="truncate max-w-[150px] min-w-0">
              <span className="text-gray-12 font-medium">{details.gitCommitMessage}</span>
            </div>
          ),
        },
        {
          icon: <User className="size-[14px] text-gray-12" iconSize="md-regular" />,
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
          icon: (
            <CircleHalfDottedClock className="size-[14px] text-gray-12" iconSize="md-regular" />
          ),
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
          icon: <Connections className="text-gray-12" iconSize="md-regular" />,
          label: "Instances",
          content: (
            <span className="text-gray-12 font-medium">{details.instances?.length ?? 0}</span>
          ),
        },
        {
          icon: <Location2 className="size-[14px] text-gray-12" iconSize="md-regular" />,
          label: "Regions",
          alignment: "start",
          content: (
            <div className="w-fit">
              <div className="gap-1 flex items-center justify-center cursor-pointer border border-grayA-3 transition-all duration-100 bg-grayA-3 p-1.5 h-[22px] rounded-md">
                {details.instances.map(r =>
                  <div className="border rounded-[10px] border-grayA-3 size-4 bg-grayA-3 flex items-center justify-center">
                    <img src={`/images/flags/${r.flagCode}.svg`} alt={r.flagCode} className="size-4" />
                  </div>
                )}
              </div>
            </div>
          ),
        },
        {
          icon: <Bolt className="size-[14px] text-gray-12" iconSize="md-regular" />,
          label: "CPU",
          content: (
            <span className="text-gray-12 font-medium">{formatCpu(details.cpuMillicores)}</span>
          ),
        },
        {
          icon: <Grid className="size-[14px] text-gray-12" iconSize="md-regular" />,
          label: "Memory",
          content: (
            <span className="text-gray-12 font-medium">{formatMemory(details.memoryMib)}</span>
          ),
        },
        {
          icon: <Harddrive className="size-[14px] text-gray-12" iconSize="md-regular" />,
          label: "Storage",
          disabled: true,
          content: <span className="text-grayA-10">—</span>,
        },
        {
          icon: <Heart className="size-[14px] text-gray-12" iconSize="md-regular" />,
          label: "Healthcheck",
          disabled: true,
          content: <span className="text-grayA-10">—</span>,
        },
        {
          icon: <ChartActivity className="size-[14px] text-gray-12" iconSize="md-regular" />,
          label: "Scaling",
          disabled: true,
          content: <span className="text-grayA-10">—</span>,
        },
      ],
    },
  ];
};
