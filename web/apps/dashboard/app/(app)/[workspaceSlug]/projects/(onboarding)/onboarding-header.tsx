"use client";
import { trpc } from "@/lib/trpc/client";
import { Check, CloudUp, Harddrive, HeartPulse, Location2, Nodes2, XMark } from "@unkey/icons";
import { useStepWizard } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useState } from "react";

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

const iconItems: { icon: React.ReactNode; large?: boolean; opacity: string }[] = [
  { icon: null, opacity: "opacity-60" },
  { icon: <Harddrive className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-75" },
  { icon: <Location2 className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-80" },
  { icon: <CloudUp className="size-9" iconSize="md-thin" />, large: true, opacity: "opacity-90" },
  { icon: <HeartPulse className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-80" },
  { icon: <Nodes2 className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-75" },
  { icon: null, opacity: "opacity-60" },
];

const IconRow = () => (
  <div
    className="p-2"
    style={{
      maskImage: "linear-gradient(to right, transparent, black 15%, black 85%, transparent)",
      WebkitMaskImage: "linear-gradient(to right, transparent, black 15%, black 85%, transparent)",
    }}
  >
    <div className="flex gap-6 items-center justify-center text-gray-12">
      {iconItems.map((item, i) => (
        <IconBox key={i} large={item.large} className={item.opacity}>
          {item.icon}
        </IconBox>
      ))}
    </div>
  </div>
);

type StepConfig = {
  title: string;
  subtitle: React.ReactNode;
  showIconRow: boolean;
};

const stepConfigs: Record<string, StepConfig> = {
  "create-project": {
    title: "Deploy your first project",
    subtitle: (
      <>
        Connect a GitHub repo and get a live URL in minutes.
        <br />
        Unkey handles builds, infra, scaling, and routing.
      </>
    ),
    showIconRow: true,
  },
  "connect-github": {
    title: "Deploy your first project",
    subtitle: (
      <>
        Connect a GitHub repo and get a live URL in minutes.
        <br />
        Unkey handles builds, infra, scaling, and routing.
      </>
    ),
    showIconRow: true,
  },
  "select-repo": {
    title: "Select a repository",
    subtitle: (
      <>
        Choose a repository and a branch containing your project.<br />
        We’ll automatically detect Dockerfiles.
      </>
    ),
    showIconRow: false,
  },

};

type OnboardingHeaderProps = {
  projectId: string | null;
};

export const OnboardingHeader = ({ projectId }: OnboardingHeaderProps) => {
  const { activeStepId } = useStepWizard();
  const [isDismissed, setIsDismissed] = useState(false);
  const config = stepConfigs[activeStepId];

  const isGithubStep = activeStepId === "connect-github";
  const { data } = trpc.github.getInstallations.useQuery(
    { projectId: projectId ?? "" },
    { enabled: isGithubStep && Boolean(projectId), staleTime: 0 },
  );
  const hasInstallations = (data?.installations?.length ?? 0) > 0;
  const showBanner = isGithubStep && hasInstallations && !isDismissed;

  if (!config) {
    return null;
  }

  return (
    <>
      {showBanner && (
        <div className="absolute top-2 left-2 right-2 rounded-[10px] p-3 gap-2.5 flex items-center shadow-[inset_0_0_0_0.75px_rgba(0,0,0,0.10)] bg-gradient-to-r from-successA-4 via-successA-1 to-success-1">
          <Check iconSize="sm-regular" />
          <div className="flex items-center gap-1">
            <span className="font-medium text-[13px] text-success-12">
              GitHub connected successfully.
            </span>
            <span className="text-[13px] text-success-12">
              You can now select a repository to deploy
            </span>
          </div>
          <button type="button" onClick={() => setIsDismissed(true)} className="ml-auto">
            <XMark iconSize="sm-regular" />
          </button>
        </div>
      )}
      <div className="flex flex-col items-center">
        {config.showIconRow && <IconRow />}
        <div className="mb-5" />
        <div className="flex flex-col items-center justify-center gap-2">
          <div className="font-semibold text-lg text-gray-12">{config.title}</div>
          <div className="text-[13px] text-gray-11 text-center">{config.subtitle}</div>
        </div>
        <div className="mb-6" />
      </div>
    </>
  );
};
