"use client";
import { BookBookmark, CloudUp, Discord, Github, Harddrive, HeartPulse, Layers3, Location2, Nodes2 } from "@unkey/icons";
import { cn } from "@unkey/ui/src/lib/utils";
import { Button } from "@unkey/ui";

type IconBoxProps = {
  children?: React.ReactNode;
  large?: boolean;
  className?: string;
};

const IconBox = ({ children, large, className }: IconBoxProps) => (
  <div
    className={cn(
      "shrink-0 flex items-center justify-center rounded-[10px] bg-transparent ring-1 ring-grayA-4 shadow-[0_2px_8px_-2px_rgba(0,0,0,0.12),0_0_0_0.75px_rgba(0,0,0,0.08)]",
      large ? "size-16" : "size-9",
      className,
    )}
  >
    {children}
  </div>
);

const items: { icon: React.ReactNode; large?: boolean; opacity: string }[] = [
  { icon: null, opacity: "opacity-60" },
  { icon: <Harddrive className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-75" },
  { icon: <Location2 className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-80" },
  { icon: <CloudUp className="size-9" iconSize="md-thin" />, large: true, opacity: "opacity-90" },
  { icon: <HeartPulse className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-80" },
  { icon: <Nodes2 className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-75" },
  { icon: null, opacity: "opacity-60" },
];

export const Onboarding = () => {
  // const state = JSON.stringify({ projectId });
  // const installUrl = `https://github.com/apps/${process.env.NEXT_PUBLIC_GITHUB_APP_NAME}/installations/new?state=${encodeURIComponent(state)}`;
  return (
    <div className="flex flex-col items-center justify-center h-screen">
      <div
        className="p-2"
        style={{
          maskImage: "linear-gradient(to right, transparent, black 15%, black 85%, transparent)",
          WebkitMaskImage:
            "linear-gradient(to right, transparent, black 15%, black 85%, transparent)",
        }}
      >
        <div className="flex gap-6 items-center justify-center text-gray-12">
          {items.map((item, i) => (
            <IconBox key={i} large={item.large} className={item.opacity}>
              {item.icon}
            </IconBox>
          ))}
        </div>
      </div>
      <div className="mb-5" />
      {/* Title-desc */}
      <div className="flex flex-col items-center justify-center gap-2">
        <div className="font-semibold text-lg text-gray-12">Deploy your first project</div>
        <div className="text-[13px] text-gray-11 text-center">
          Connect a GitHub repo and get a live URL in minutes.
          <br />
          Unkey handles builds, infra, scaling, and routing.
        </div>
      </div>
      <div className="mb-6" />
      <div className="border border-grayA-5 rounded-[14px] flex  justify-center items-center gap-4 py-[18px] px-4">
        <div className="size-8 rounded-[10px] bg-gray-12 grid place-items-center">
          <Layers3 className="size-[18px] text-gray-1" iconSize="md-medium" />
        </div>
        <div className="flex flex-col gap-3">
          <span className="font-medium text-gray-12 text-[13px] leading-[9px]">Import project</span>
          <span className="text-gray-10 text-[13px]  leading-[9px]">Add a repo from your GitHub account</span>
        </div>
        <Button variant="outline" className="ml-20 rounded-lg border-grayA-4 hover:bg-grayA-2 shadow-sm hover:shadow-md transition-all">
          <Github className="!size-[18px] text-gray-12 shrink-0" />
          <a href={""} className="text-sm text-gray-12 font-medium" target="_blank" rel="noopener noreferrer">
            Import from GitHub
          </a>
        </Button>
      </div>
      <div className="mb-7" />
      <div className="flex gap-3 items-center">
        <Button
          variant="outline"
          className="text-gray-12 text-[13px] font-medium border border-grayA-4 gap-2 rounded-full flex items-center px-3 py-1.5 transition-all"
          onClick={() => window.open("https://www.unkey.com/docs/introduction", "_blank", "noopener,noreferrer")}
        >
          <BookBookmark className="text-gray-12 shrink-0 size-[18px]" iconSize="sm-regular" />
          View documentation
        </Button>
        <Button
          variant="outline"
          className="text-gray-12 text-[13px] font-medium border border-grayA-4 gap-2 rounded-full flex items-center px-3 py-1.5 transition-all"
          onClick={() => window.open("https://discord.gg/fDbezjbJbD", "_blank", "noopener,noreferrer")}
        >
          <div className="size-[18px] overflow-hidden flex items-center justify-center">
            <Discord className="text-feature-11 shrink-0" style={{ width: 18, height: 18 }} iconSize="sm-regular" />
          </div>
          Join community
        </Button>
      </div>
    </div>
  )
};
