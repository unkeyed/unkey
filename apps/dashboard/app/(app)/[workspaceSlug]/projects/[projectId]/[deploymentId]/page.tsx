"use client";

import {
  Bolt,
  ChartActivity,
  CircleCheck,
  Focus,
  Heart,
  Layers3,
} from "@unkey/icons";
import { TreeConnectionLine, TreeLayout } from "./components/unkey-flow";
import { InfiniteCanvas } from "./components/unkey-flow/components/canvas/infinite-canvas";
// biome-ignore lint/style/useImportType: <explanation>
import { TreeNode } from "./components/unkey-flow/types";
import { cn } from "@unkey/ui/src/lib/utils";

const buildParentMap = (
  node: TreeNode,
  parentMap = new Map<string, TreeNode>()
): Map<string, TreeNode> => {
  if (node.children) {
    for (const child of node.children) {
      parentMap.set(child.id, node);
      buildParentMap(child, parentMap);
    }
  }
  return parentMap;
};

const IngressNode = ({ node }: { node: TreeNode }) => (
  <div className="w-[70px] h-[20px] ring-4 rounded-full ring-grayA-5 bg-gray-9  flex items-center justify-center p-2.5 shadow-sm">
    <div className="font-mono text-[9px] font-medium  text-white leading-[6px]">
      {node.label}
    </div>
  </div>
);

const RegionNode = ({ node }: { node: TreeNode }) => {
  const gradientColorMap: Record<string, string> = {
    "us-east-1": "hsl(var(--infoA-3))",
    "ap-east-1": "hsl(var(--errorA-3))",
    "ap-south-1": "hsl(var(--orangeA-3))",
    "eu-west-1": "hsl(var(--infoA-3))",
  };

  const gradientColor = gradientColorMap[node.id] ?? "hsl(var(--grayA-3))";

  return (
    <div
      className="w-[282px] h-[100px] border border-grayA-4 rounded-[14px] flex flex-col bg-white dark:bg-black shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)] hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_hsl(var(--grayA-3))]
hover:ring-2 hover:ring-grayA-2 hover:scale-[1.02] transition-all duration-200 cursor-pointer hover:ring-offset-0"
    >
      <div
        className="border-b border-grayA-4 flex px-3 py-2.5 rounded-t-[14px]"
        style={{
          background: `radial-gradient(circle at 5% 15%, ${gradientColor} 0%, transparent 20%), light-dark(#FFF, #000)`,
        }}
      >
        <div className="flex items-center justify-between gap-3">
          <div className="border rounded-[10px] border-grayA-3 size-9 bg-grayA-3 flex items-center justify-center">
            {node.metadata.flagComponent}
          </div>
          <div className="flex flex-col gap-[3px] justify-center h-9 py-2">
            <div className="text-accent-12 font-medium text-xs font-mono">
              {node.label}
            </div>
            <div className="text-gray-9 text-[11px]">
              {node.metadata.zones} availability{" "}
              {node.metadata.zones.length > 1 ? "zones" : "zone"}
            </div>
          </div>
        </div>
        <div className="flex gap-2 items-center ml-auto">
          <div className="border bg-gray-1 border-grayA-3 h-11 rounded-lg w-8">
            <div className="h-6 border-b border-grayA-3 relative">
              <StatusDot variant="success" />
            </div>
            <div className="h-5 bg-grayA-2 pl-1 pt-[3px]">
              <CircleCheck className="text-gray-9" iconSize="sm-regular" />
            </div>
          </div>
          <div className="border bg-gray-1 border-grayA-3 h-11 rounded-lg w-8 ml-auto">
            <div className="h-6 border-b border-grayA-3 relative">
              <StatusDot variant="info" />
            </div>
            <div className="h-5 bg-grayA-2 pl-1 pt-[3px]">
              <Heart className="text-gray-9" iconSize="sm-regular" />
            </div>
          </div>
        </div>
      </div>
      <div className="p-1 flex items-center h-full bg-grayA-2 rounded-b-[14px]">
        <div className="bg-grayA-3 p-1.5 flex items-center justify-between rounded-full h-5 gap-1.5">
          <ChartActivity iconSize="sm-medium" className="shrink-0" />
          <span className="text-gray-9 text-[10px] tabular-nums">
            {node.metadata.instances}
          </span>
        </div>
        <div className="flex items-center gap-2 ml-auto">
          <div className="bg-grayA-3 p-1.5 flex items-center justify-between rounded-full h-5 gap-1.5">
            <Bolt iconSize="sm-medium" className="shrink-0" />
            <span className="text-gray-9 text-[10px] tabular-nums">
              {node.metadata.power}%
            </span>
          </div>
          <div className="bg-grayA-3 p-1.5 flex items-center justify-between rounded-full h-5 gap-1.5">
            <Focus iconSize="sm-regular" className="shrink-0" />
            <span className="text-gray-9 text-[10px] tabular-nums">
              {node.metadata.storage}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
};

const InstanceNode = ({
  node,
  parent,
}: {
  node: TreeNode;
  parent?: TreeNode;
}) => {
  const regionColorMap: Record<
    string,
    { bg: string; text: string; glow: string }
  > = {
    "us-east-1": {
      bg: "bg-blueA-2",
      text: "text-blue-10",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_color-mix(in_srgb,hsl(var(--blueA-9))_15%,transparent)]",
    },
    "ap-east-1": {
      bg: "bg-redA-2",
      text: "text-red-10",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_color-mix(in_srgb,hsl(var(--redA-9))_15%,transparent)]",
    },
    "ap-south-1": {
      bg: "bg-orangeA-2",
      text: "text-orange-10",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_color-mix(in_srgb,hsl(var(--orangeA-9))_15%,transparent)]",
    },
    "eu-west-1": {
      bg: "bg-blueA-2",
      text: "text-blue-10",
      glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_color-mix(in_srgb,hsl(var(--blueA-9))_15%,transparent)]",
    },
  };

  const colors = parent?.id
    ? regionColorMap[parent.id] ?? {
        bg: "bg-grayA-2",
        text: "text-gray-11",
        glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_hsl(var(--grayA-3))]",
      }
    : {
        bg: "bg-grayA-2",
        text: "text-gray-11",
        glow: "hover:shadow-[0_4px_16px_-4px_rgba(0,0,0,0.15),0_0_0_1px_hsl(var(--grayA-6)),0_0_30px_hsl(var(--grayA-3))]",
      };

  return (
    <div
      className={cn(
        "w-[282px] h-[100px] border border-grayA-4 rounded-[14px] flex flex-col bg-white dark:bg-black shadow-[0_2px_8px_-2px_rgba(0,0,0,0.1)]",
        colors.glow,
        "hover:ring-2 hover:ring-grayA-2 hover:scale-[1.02] transition-all duration-200 cursor-pointer hover:ring-offset-0"
      )}
    >
      <div className="border-b border-grayA-4 flex px-3 py-2.5 rounded-t-[14px]">
        <div className="flex items-center justify-between gap-3">
          <div
            className={cn(
              "border rounded-[10px] size-9 flex items-center justify-center border-grayA-5",
              colors.bg
            )}
          >
            <Layers3 iconSize="sm-medium" className={colors.text} />
          </div>
          <div className="flex flex-col gap-[3px] justify-center h-9 py-2">
            <div className="text-accent-12 font-medium text-xs font-mono">
              {node.label}
            </div>
            <div className="text-gray-9 text-[11px]">
              {node.metadata.description}
            </div>
          </div>
        </div>
        <div className="flex gap-2 items-center ml-auto">
          <div className="border bg-gray-1 border-grayA-3 h-11 rounded-lg w-8">
            <div className="h-6 border-b border-grayA-3 relative">
              <StatusDot variant="success" />
            </div>
            <div className="h-5 bg-grayA-2 pl-1 pt-[3px]">
              <CircleCheck className="text-gray-9" iconSize="sm-regular" />
            </div>
          </div>
          <div className="border bg-gray-1 border-grayA-3 h-11 rounded-lg w-8 ml-auto">
            <div className="h-6 border-b border-grayA-3 relative">
              <StatusDot variant="info" />
            </div>
            <div className="h-5 bg-grayA-2 pl-1 pt-[3px]">
              <Heart className="text-gray-9" iconSize="sm-regular" />
            </div>
          </div>
        </div>
      </div>
      <div className="p-1 flex items-center h-full bg-grayA-2 rounded-b-[14px]">
        <div className="size-[22px] bg-grayA-3 rounded-full p-[3px] flex items-center justify-center mr-1.5">
          {parent?.metadata.flagComponent}
        </div>
        <div className="bg-grayA-3 p-1.5 flex items-center justify-between rounded-full h-5 gap-1.5">
          <ChartActivity iconSize="sm-medium" className="shrink-0" />
          <span className="text-gray-9 text-[10px] tabular-nums">
            {node.metadata.instances}
          </span>
        </div>
        <div className="flex items-center gap-2 ml-auto">
          <div className="bg-grayA-3 p-1.5 flex items-center justify-between rounded-full h-5 gap-1.5">
            <Bolt iconSize="sm-medium" className="shrink-0" />
            <span className="text-gray-9 text-[10px] tabular-nums">
              {node.metadata.power}%
            </span>
          </div>
          <div className="bg-grayA-3 p-1.5 flex items-center justify-between rounded-full h-5 gap-1.5">
            <Focus iconSize="sm-regular" className="shrink-0" />
            <span className="text-gray-9 text-[10px] tabular-nums">
              {node.metadata.storage}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
};

const DefaultNode = ({ node }: { node: TreeNode }) => (
  <div className="w-[500px] h-[70px] border border-grayA-4 rounded-[14px] bg-gray-1 flex items-center justify-center">
    {node.label}
  </div>
);

// Object lookup map
const nodeComponents = {
  ingress: IngressNode,
  region: RegionNode,
  instance: InstanceNode,
} as const;

export default function DeploymentDetailsPage() {
  const parentMap = buildParentMap(deploymentTree);

  return (
    <InfiniteCanvas>
      <TreeLayout
        data={deploymentTree}
        nodeSpacing={{ x: 25, y: 150 }}
        renderNode={(node) => {
          const parent = parentMap.get(node.id);
          const NodeComponent =
            nodeComponents[node.metadata.type as keyof typeof nodeComponents] ??
            DefaultNode;
          return <NodeComponent node={node} parent={parent} />;
        }}
        renderConnection={(from, to, parent, child) => {
          const childIndex =
            parent.children?.findIndex((c) => c.id === child.id) ?? 0;
          const childCount = parent.children?.length ?? 1;
          const xOffset = (childIndex - (childCount - 1) / 2) * 5;

          return (
            <TreeConnectionLine
              key={`${parent.id}-${child.id}`}
              from={{ x: from.x + xOffset, y: from.y }}
              to={{ x: to.x, y: to.y }}
            />
          );
        }}
      />
    </InfiniteCanvas>
  );
}

const colorMap = {
  success: {
    bg: "bg-success-9",
    ring: "hsl(var(--successA-4))",
  },
  info: {
    bg: "bg-info-9",
    ring: "hsl(var(--infoA-4))",
  },
} as const;

type StatusDotProps = {
  variant: keyof typeof colorMap;
};

const StatusDot = ({ variant }: StatusDotProps) => {
  const { bg, ring } = colorMap[variant];
  return (
    <>
      <div className="absolute top-1.5 right-1.5 size-[7px]">
        {/* Ring 1 */}
        <div
          className="absolute inset-0 rounded-full"
          style={{
            animation: "breathe-ring 2s ease-in-out infinite",
            boxShadow: `0 0 0 1.5px ${ring}`,
          }}
        />
        {/* Ring 2 */}
        <div
          className="absolute inset-0 rounded-full"
          style={{
            animation: "breathe-ring 2s ease-in-out infinite 1s",
            boxShadow: `0 0 0 1.5px ${ring}`,
          }}
        />
        {/* Solid dot */}
        <div className={cn("absolute inset-0 rounded-full", bg)} />
      </div>
      <style>{`
        @keyframes breathe-ring {
          0%, 100% {
            transform: scale(1);
            opacity: 0.6;
          }
          50% {
            transform: scale(2.2);
            opacity: 0;
          }
        }
      `}</style>
    </>
  );
};

const UsFlag = () => (
  <svg
    width="16"
    height="16"
    viewBox="0 0 16 16"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <g clip-path="url(#clip0_35416_18049)">
      <mask
        id="mask0_35416_18049"
        style={{ maskType: "luminance" }}
        maskUnits="userSpaceOnUse"
        x="0"
        y="0"
        width="16"
        height="16"
      >
        <path
          d="M8 16C12.4183 16 16 12.4183 16 8C16 3.58172 12.4183 0 8 0C3.58172 0 0 3.58172 0 8C0 12.4183 3.58172 16 8 16Z"
          fill="white"
        />
      </mask>
      <g mask="url(#mask0_35416_18049)">
        <path
          d="M8 0H16V2L15 3L16 4V6L15 7L16 8V10L15 11L16 12V14L8 15L0 14V12L1 11L0 10V8L8 0Z"
          fill="#EEEEEE"
        />
        <path
          d="M7 2H16V4H7V2ZM7 6H16V8H8L7 6ZM0 10H16V12H0V10ZM0 14H16V16H0V14Z"
          fill="#D80027"
        />
        <path d="M0 0H8V8H0V0Z" fill="#0052B4" />
        <path
          d="M5.84375 7.59375L7.625 6.3125H5.4375L7.21875 7.59375L6.53125 5.5L5.84375 7.59375ZM3.3125 7.59375L5.09375 6.3125H2.90625L4.6875 7.59375L4 5.5L3.3125 7.59375ZM0.78125 7.59375L2.5625 6.3125H0.375L2.15625 7.59375L1.46875 5.5L0.78125 7.59375ZM5.84375 5.0625L7.625 3.78125H5.4375L7.21875 5.0625L6.53125 2.96875L5.84375 5.0625ZM3.3125 5.0625L5.09375 3.78125H2.90625L4.6875 5.0625L4 2.96875L3.3125 5.0625ZM0.78125 5.0625L2.5625 3.78125H0.375L2.15625 5.0625L1.46875 2.96875L0.78125 5.0625ZM5.84375 2.5L7.625 1.21875H5.4375L7.21875 2.5L6.53125 0.40625L5.84375 2.5ZM3.3125 2.5L5.09375 1.21875H2.90625L4.6875 2.5L4 0.40625L3.3125 2.5ZM0.78125 2.5L2.5625 1.21875H0.375L2.15625 2.5L1.46875 0.40625L0.78125 2.5Z"
          fill="#EEEEEE"
        />
      </g>
    </g>
    <circle cx="8" cy="8" r="7.5" stroke="black" stroke-opacity="0.2" />
    <defs>
      <clipPath id="clip0_35416_18049">
        <rect width="16" height="16" fill="white" />
      </clipPath>
    </defs>
  </svg>
);

const IndianFlag = () => (
  <svg
    width="16"
    height="16"
    viewBox="0 0 16 16"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <g clip-path="url(#clip0_35416_18940)">
      <mask
        id="mask0_35416_18940"
        style={{ maskType: "luminance" }}
        maskUnits="userSpaceOnUse"
        x="0"
        y="0"
        width="16"
        height="16"
      >
        <path
          d="M8 16C12.4183 16 16 12.4183 16 8C16 3.58172 12.4183 0 8 0C3.58172 0 0 3.58172 0 8C0 12.4183 3.58172 16 8 16Z"
          fill="white"
        />
      </mask>
      <g mask="url(#mask0_35416_18940)">
        <path d="M0 5L8 4L16 5V11L8 12L0 11V5Z" fill="#EEEEEE" />
        <path d="M0 0H16V5H0V0Z" fill="#FF9811" />
        <path d="M0 11H16V16H0V11Z" fill="#6DA544" />
        <path
          d="M8 10.25C9.24264 10.25 10.25 9.24264 10.25 8C10.25 6.75736 9.24264 5.75 8 5.75C6.75736 5.75 5.75 6.75736 5.75 8C5.75 9.24264 6.75736 10.25 8 10.25Z"
          fill="#0052B4"
        />
        <path
          d="M8 9.5C8.82843 9.5 9.5 8.82843 9.5 8C9.5 7.17157 8.82843 6.5 8 6.5C7.17157 6.5 6.5 7.17157 6.5 8C6.5 8.82843 7.17157 9.5 8 9.5Z"
          fill="#EEEEEE"
        />
        <path
          d="M8 8.75C8.41421 8.75 8.75 8.41421 8.75 8C8.75 7.58579 8.41421 7.25 8 7.25C7.58579 7.25 7.25 7.58579 7.25 8C7.25 8.41421 7.58579 8.75 8 8.75Z"
          fill="#0052B4"
        />
      </g>
    </g>
    <circle cx="8" cy="8" r="7.5" stroke="black" stroke-opacity="0.2" />
    <defs>
      <clipPath id="clip0_35416_18940">
        <rect width="16" height="16" fill="white" />
      </clipPath>
    </defs>
  </svg>
);

const TurkishFlag = () => (
  <svg
    width="16"
    height="16"
    viewBox="0 0 16 16"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <g clipPath="url(#clip0_35442_86063)">
      <mask
        id="mask0_35442_86063"
        style={{ maskType: "luminance" }}
        maskUnits="userSpaceOnUse"
        x="0"
        y="0"
        width="16"
        height="16"
      >
        <path
          d="M8 16C12.4183 16 16 12.4183 16 8C16 3.58172 12.4183 0 8 0C3.58172 0 0 3.58172 0 8C0 12.4183 3.58172 16 8 16Z"
          fill="white"
        />
      </mask>
      <g mask="url(#mask0_35442_86063)">
        <path d="M0 0H16V16H0V0Z" fill="#D80027" />
        <path
          d="M8.82497 6.05312C8.64372 6.80937 8.32185 6.66562 8.16247 7.325C7.72175 7.20972 7.34378 6.92626 7.1097 6.53546C6.87561 6.14465 6.80406 5.67765 6.9104 5.23469C7.01674 4.79173 7.29251 4.40811 7.6785 4.16617C8.06449 3.92423 8.52995 3.84325 8.97497 3.94062C8.65935 5.25937 8.98747 5.37812 8.82497 6.05312ZM6.40622 6.6125C7.06872 7.01875 6.83122 7.28125 7.40935 7.6375C7.15985 8.0127 6.77458 8.27653 6.33455 8.37351C5.89453 8.47048 5.43404 8.39305 5.04994 8.15749C4.66584 7.92192 4.38804 7.54659 4.27499 7.11042C4.16193 6.67425 4.22241 6.21124 4.44372 5.81875C5.59997 6.52812 5.81247 6.25 6.40622 6.6125ZM6.18747 9.09062C6.78122 8.58437 6.95935 8.89062 7.47497 8.45312C7.75105 8.80679 7.8799 9.25339 7.83464 9.69977C7.78938 10.1461 7.57351 10.5578 7.23207 10.8488C6.89062 11.1399 6.44998 11.2879 6.00207 11.2619C5.55417 11.2359 5.13359 11.038 4.8281 10.7094C5.85935 9.82812 5.65935 9.54062 6.1906 9.09062H6.18747ZM8.48122 10.0594C8.18122 9.34062 8.5281 9.26875 8.26872 8.64375C8.68972 8.49203 9.15296 8.50829 9.56229 8.68913C9.97161 8.86998 10.2956 9.20152 10.4669 9.61492C10.6382 10.0283 10.6437 10.4918 10.4823 10.9092C10.3208 11.3266 10.0049 11.6657 9.59997 11.8562C9.08122 10.6062 8.74685 10.7031 8.48122 10.0625V10.0594ZM10.1125 8.18437C9.33435 8.24687 9.37185 7.89375 8.69685 7.94687C8.67629 7.49555 8.83202 7.05392 9.13112 6.71532C9.43022 6.37672 9.84926 6.16768 10.2997 6.13238C10.7501 6.09707 11.1966 6.23828 11.5448 6.52614C11.893 6.81401 12.1156 7.22599 12.1656 7.675C10.8125 7.78125 10.8031 8.12812 10.1125 8.18437Z"
          fill="#EEEEEE"
        />
      </g>
    </g>
    <circle cx="8" cy="8" r="7.5" stroke="black" strokeOpacity="0.2" />
    <defs>
      <clipPath id="clip0_35442_86063">
        <rect width="16" height="16" fill="white" />
      </clipPath>
    </defs>
  </svg>
);

const EuFlag = () => (
  <svg
    width="16"
    height="16"
    viewBox="0 0 16 16"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
  >
    <g clipPath="url(#clip0_eu_flag)">
      <mask
        id="mask0_eu_flag"
        style={{ maskType: "luminance" }}
        maskUnits="userSpaceOnUse"
        x="0"
        y="0"
        width="16"
        height="16"
      >
        <path
          d="M8 16C12.4183 16 16 12.4183 16 8C16 3.58172 12.4183 0 8 0C3.58172 0 0 3.58172 0 8C0 12.4183 3.58172 16 8 16Z"
          fill="white"
        />
      </mask>
      <g mask="url(#mask0_eu_flag)">
        <path d="M0 0H16V16H0V0Z" fill="#039" />
        <g fill="#fc0">
          <path d="M8 2.5L8.2 3.1 8.8 3.1 8.3 3.5 8.5 4.1 8 3.7 7.5 4.1 7.7 3.5 7.2 3.1 7.8 3.1z" />
          <path d="M11.2 3.5L11.4 4.1 12 4.1 11.5 4.5 11.7 5.1 11.2 4.7 10.7 5.1 10.9 4.5 10.4 4.1 11 4.1z" />
          <path d="M12.5 6L12.7 6.6 13.3 6.6 12.8 7 13 7.6 12.5 7.2 12 7.6 12.2 7 11.7 6.6 12.3 6.6z" />
          <path d="M12.5 9.4L12.7 10 13.3 10 12.8 10.4 13 11 12.5 10.6 12 11 12.2 10.4 11.7 10 12.3 10z" />
          <path d="M11.2 11.9L11.4 12.5 12 12.5 11.5 12.9 11.7 13.5 11.2 13.1 10.7 13.5 10.9 12.9 10.4 12.5 11 12.5z" />
          <path d="M8 13.5L8.2 14.1 8.8 14.1 8.3 14.5 8.5 15.1 8 14.7 7.5 15.1 7.7 14.5 7.2 14.1 7.8 14.1z" />
          <path d="M4.8 11.9L5 12.5 5.6 12.5 5.1 12.9 5.3 13.5 4.8 13.1 4.3 13.5 4.5 12.9 4 12.5 4.6 12.5z" />
          <path d="M3.5 9.4L3.7 10 4.3 10 3.8 10.4 4 11 3.5 10.6 3 11 3.2 10.4 2.7 10 3.3 10z" />
          <path d="M3.5 6L3.7 6.6 4.3 6.6 3.8 7 4 7.6 3.5 7.2 3 7.6 3.2 7 2.7 6.6 3.3 6.6z" />
          <path d="M4.8 3.5L5 4.1 5.6 4.1 5.1 4.5 5.3 5.1 4.8 4.7 4.3 5.1 4.5 4.5 4 4.1 4.6 4.1z" />
        </g>
      </g>
    </g>
    <circle cx="8" cy="8" r="7.5" stroke="black" strokeOpacity="0.2" />
    <defs>
      <clipPath id="clip0_eu_flag">
        <rect width="16" height="16" fill="white" />
      </clipPath>
    </defs>
  </svg>
);

const deploymentTree: TreeNode = {
  id: "ingress",
  label: "INTERNET",
  metadata: {
    type: "ingress",
  },
  children: [
    {
      id: "us-east-1",
      label: "us-east-1",
      metadata: {
        type: "region",
        description: "3 availability zones",
        zones: 2,
        instances: 28,
        replicas: 2,
        power: 31,
        storage: "768mi",
        bandwidth: "1gb",
        latency: "2.4ms",
        status: "active",
        health: "healthy",
        flagComponent: <UsFlag />,
      },
      children: [
        {
          id: "us-east-1-gw-9h4k-1",
          label: "gw-9h4k",
          metadata: {
            type: "instance",
            description: "Instance replica",
            cpu: "47%",
            memory: "26%",
            instances: 28,
            replicas: 2,
            power: 31,
            storage: "768mi",
            bandwidth: "1gb",
            latency: "2.4ms",
            status: "active",
            health: "healthy",
          },
        },
        {
          id: "us-east-1-gw-9h4k-1-123213",
          label: "gw-9hasdsad4k",
          metadata: {
            type: "instance",
            description: "Instance replica",
            replicas: 2,
            power: "38%",
            cpu: "47%",
            memory: "26%",
            latency: "6ms",
            status: "active",
            health: "healthy",
          },
        },
      ],
    },
    {
      id: "ap-east-1",
      label: "ap-east-1",
      metadata: {
        type: "region",
        description: "2 availability zones",
        zones: 1,
        instances: 24,
        replicas: 2,
        power: 27,
        storage: "512mi",
        bandwidth: "1gb",
        latency: "3.1ms",
        status: "active",
        health: "healthy",
        flagComponent: <TurkishFlag />,
      },
      children: [
        {
          id: "ap-east-1-gw-7f2c-1",
          label: "gw-7f2c",
          metadata: {
            type: "instance",
            instances: 24,
            description: "Instance replica",
            storage: "512mi",
            replicas: 2,
            power: "35%",
            cpu: "44%",
            memory: "22%",
            latency: "9ms",
            status: "active",
            health: "healthy",
          },
        },
      ],
    },
    {
      id: "ap-south-1",
      label: "ap-south-1",
      metadata: {
        type: "region",
        description: "3 availability zones",
        zones: 2,
        instances: 24,
        replicas: 2,
        power: 23,
        storage: "512mi",
        bandwidth: "1gb",
        latency: "3.1ms",
        status: "active",
        health: "healthy",
        flagComponent: <IndianFlag />,
      },
      children: [
        {
          id: "ap-south-1-gw-8k3d-1",
          label: "gw-8k3d",
          metadata: {
            type: "instance",
            description: "Instance replica",
            replicas: 2,
            power: "41%",
            cpu: "52%",
            memory: "28%",
            latency: "7ms",
            status: "active",
            health: "healthy",
          },
        },
        {
          id: "ap-south-1-gw-8k3d-2",
          label: "gw-8k3d",
          metadata: {
            type: "instance",
            description: "Instance replica",
            instances: 24,
            replicas: 2,
            power: "19%",
            cpu: "31%",
            memory: "15%",
            latency: "5.2ms",
            status: "active",
            health: "healthy",
          },
        },
      ],
    },
    {
      id: "eu-west-1",
      label: "eu-west-1",
      metadata: {
        type: "region",
        description: "3 availability zones",
        zones: 1,
        instances: 32,
        replicas: 2,
        power: 31,
        storage: "1gi",
        bandwidth: "1gb",
        latency: "2.8ms",
        status: "active",
        health: "healthy",
        flagComponent: <EuFlag />,
      },
      children: [
        {
          id: "eu-west-1-gw-2m9p-2",
          label: "gw-2m9p",
          metadata: {
            type: "instance",
            description: "Instance replica",
            instances: 32,
            replicas: 2,
            power: "31%",
            cpu: "45%",
            memory: "21%",
            latency: "5.8ms",
            status: "active",
            health: "healthy",
          },
        },
      ],
    },
  ],
};
