"use client";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { Loader2 } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useTransition } from "react";
export type NavItem = {
  disabled?: boolean;
  tooltip?: string;
  icon: React.ReactNode;
  href: string;
  external?: boolean;
  label: string;
  active?: boolean;
  tag?: React.ReactNode;
};
export const NavLink: React.FC<{ item: NavItem }> = ({ item }) => {
  const [isPending, startTransition] = useTransition();
  const router = useRouter();
  const link = (
    <Link
      prefetch
      href={item.href}
      onClick={() => {
        if (!item.external) {
          startTransition(() => {
            router.push(item.href);
          });
        }
      }}
      target={item.external ? "_blank" : undefined}
      className={cn(
        "group flex gap-x-2 rounded-md px-2 py-1 text-sm  font-medium leading-6 items-center hover:bg-gray-200 dark:hover:bg-gray-800 justify-between",
        {
          "bg-gray-200 dark:bg-gray-800": item.active,
          "text-content-subtle pointer-events-none": item.disabled,
        },
      )}
    >
      <div className="flex group gap-x-2">
        <span className="text-content-subtle border-border group-hover:shadow  flex h-6 w-6 shrink-0 items-center justify-center rounded-lg border text-[0.625rem] font-medium bg-white">
          {isPending ? <Loader2 className="w-4 h-4 shrink-0 animate-spin" /> : item.icon}
        </span>
        <p className="truncate whitespace-nowrap">{item.label}</p>
      </div>
      {item.tag}
    </Link>
  );

  if (item.tooltip) {
    return (
      <Tooltip>
        <TooltipTrigger className="w-full">
          {link}
          <TooltipContent>{item.tooltip}</TooltipContent>
        </TooltipTrigger>
      </Tooltip>
    );
  }
  return link;
};
