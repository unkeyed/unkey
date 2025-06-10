"use client";

import Link from "next/link";
import * as React from "react";

import { cn } from "@/lib/utils";
import { Separator } from "@unkey/ui";
import { useRouter, useSelectedLayoutSegment } from "next/navigation";

type Props = {
  navigation: {
    label: string;
    href: string;
    segment: string | null;
    tag?: string;
    isActive?: boolean;
  }[];
  segment?: string;
  className?: string;
};

export const Navbar: React.FC<React.PropsWithChildren<Props>> = ({
  navigation,
  className,
  segment,
}) => {
  return (
    <nav className={cn("sticky top-0 bg-background", className)}>
      <div className="flex items-center w-full pl-2 overflow-x-auto">
        <ul className="flex flex-row gap-4">
          {navigation.map(({ label, href, segment: _segment, tag }) => (
            <NavItem
              key={label}
              label={label}
              href={href}
              segment={_segment}
              tag={tag}
              isActive={segment === _segment}
            />
          ))}
        </ul>
      </div>
      <Separator className="bg-gray-4" />
    </nav>
  );
};

const NavItem: React.FC<Props["navigation"][0]> = ({ label, href, segment, tag, isActive }) => {
  const selectedSegment = useSelectedLayoutSegment();
  const [isPending, startTransition] = React.useTransition();
  const router = useRouter();

  const active = segment === selectedSegment || isActive;

  return (
    <li
      className={cn("flex shrink-0 list-none border-b-2 border-transparent p-2", {
        "border-primary ": active,
        "animate-pulse": isPending,
      })}
    >
      <Link
        prefetch
        href={href}
        onClick={() =>
          startTransition(() => {
            router.push(href);
          })
        }
        className={cn(
          "text-sm flex items-center gap-1 font-medium px-3 -mx-3 text-content-subtle  hover:bg-background-subtle rounded-md hover:text-primary",
          {
            "text-primary": active,
          },
        )}
      >
        {label}
        {tag ? (
          <div className="bg-background border text-content-subtle rounded text-xs px-1 py-0.5  font-mono ">
            {tag}
          </div>
        ) : null}
      </Link>
    </li>
  );
};
