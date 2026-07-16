"use client";

import { cn } from "@/lib/utils";
import { Earth, Key, ShieldKey, User, WindowLayout } from "@unkey/icons";
import { Button } from "@unkey/ui";
import type { ReactNode } from "react";

type IconBoxProps = {
  children?: ReactNode;
  large?: boolean;
  className?: string;
};

const IconBox = ({ children, large, className }: IconBoxProps) => (
  <div
    className={cn(
      "shrink-0 flex items-center justify-center rounded-[10px] bg-transparent ring-1 ring-grayA-4 shadow-sm shadow-grayA-8/20 dark:shadow-none",
      large ? "size-16" : "size-9",
      className,
    )}
  >
    {children}
  </div>
);

const flankItems: { icon: ReactNode; large?: boolean; opacity: string }[] = [
  { icon: <Earth className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-50" },
  { icon: <User className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-75" },
  {
    icon: <WindowLayout className="size-9" iconSize="md-thin" />,
    large: true,
    opacity: "opacity-90",
  },
  { icon: <Key className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-75" },
  { icon: <ShieldKey className="size-[18px]" iconSize="md-medium" />, opacity: "opacity-50" },
];

const PortalIconRow = () => (
  <div
    aria-hidden="true"
    className="p-2 mb-8"
    style={{
      maskImage: "linear-gradient(to right, transparent, black 20%, black 80%, transparent)",
      WebkitMaskImage: "linear-gradient(to right, transparent, black 20%, black 80%, transparent)",
    }}
  >
    <div className="flex gap-6 items-center justify-center text-gray-12">
      {flankItems.map((item, i) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: static row, index is stable
        <IconBox key={i} large={item.large} className={item.opacity}>
          {item.icon}
        </IconBox>
      ))}
    </div>
  </div>
);

export function SetupHero({ enabling, onEnable }: { enabling: boolean; onEnable: () => void }) {
  return (
    <div className="flex w-full justify-center rounded-lg border border-grayA-4 p-12">
      <div className="flex flex-col items-center text-center">
        <PortalIconRow />

        <h2 className="text-accent-12 font-semibold text-2xl leading-8 mb-1">Customer portal</h2>
        <p className="text-accent-11 text-sm leading-6 max-w-md text-balance mb-6">
          An Unkey-hosted portal that allows your customers to manage their keys themselves.
        </p>

        <Button
          variant="primary"
          size="md"
          loading={enabling}
          loadingLabel="Enabling customer portal"
          onClick={onEnable}
        >
          Enable Customer portal
        </Button>
      </div>
    </div>
  );
}
