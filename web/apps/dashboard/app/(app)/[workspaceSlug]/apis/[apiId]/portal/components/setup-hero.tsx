"use client";

import { cn } from "@/lib/utils";
import { Button } from "@unkey/ui";
import {
  IconEarthOutline18,
  IconKeyOutline18,
  IconShieldKeyOutline18,
  IconUserOutline18,
  IconWindowLayoutOutline18,
} from "nucleo-ui-outline-18";
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
  { icon: <IconEarthOutline18 />, opacity: "opacity-50" },
  { icon: <IconUserOutline18 />, opacity: "opacity-75" },
  {
    icon: <IconWindowLayoutOutline18 className="size-9" />,
    large: true,
    opacity: "opacity-90",
  },
  { icon: <IconKeyOutline18 />, opacity: "opacity-75" },
  { icon: <IconShieldKeyOutline18 />, opacity: "opacity-50" },
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

export function SetupHero({ onEnable }: { onEnable: () => void }) {
  return (
    <div className="flex w-full justify-center rounded-lg border border-grayA-4 p-12">
      <div className="flex flex-col items-center text-center">
        <PortalIconRow />

        <h2 className="text-accent-12 font-semibold text-2xl leading-8 mb-1">Customer portal</h2>
        <p className="text-accent-11 text-sm leading-6 max-w-md text-balance mb-6">
          An Unkey-hosted portal that allows your customers to manage their keys themselves.
        </p>

        <Button variant="primary" size="md" onClick={onEnable}>
          Enable Customer portal
        </Button>
      </div>
    </div>
  );
}
