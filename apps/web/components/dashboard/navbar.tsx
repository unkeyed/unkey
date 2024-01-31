"use client";

import Link from "next/link";
import * as React from "react";

import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";
import { useRouter, useSelectedLayoutSegment } from "next/navigation";

type Props = {
  navigation: { label: string; href: string; segment: string | null }[];
  className?: string;
};

export const Navbar: React.FC<React.PropsWithChildren<Props>> = ({ navigation, className }) => {
  return (
    <nav className={cn("sticky top-0 bg-background", className)}>
      <div className="flex items-center w-full pl-1 overflow-x-auto">
        <ul className="flex flex-row gap-4">
          {navigation.map(({ label, href, segment }) => (
            <NavItem key={label} label={label} href={href} segment={segment} />
          ))}
        </ul>
      </div>
      <Separator />
    </nav>
  );
};

const NavItem: React.FC<Props["navigation"][0]> = ({ label, href, segment }) => {
  const selectedSegment = useSelectedLayoutSegment();
  const [isPending, startTransition] = React.useTransition();
  const router = useRouter();

  const active = segment === selectedSegment;
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
          "text-sm font-medium py-2 px-3 -mx-3 text-content-subtle  hover:bg-background-subtle rounded-md hover:text-primary",
          {
            "text-primary": active,
          },
        )}
      >
        {label}
      </Link>
    </li>
  );
};
