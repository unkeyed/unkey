"use client";
import { cn } from "@/lib/utils";
import Link from "next/link";
import { useSelectedLayoutSegment } from "next/navigation";

type Props = { href: string; label: string };

export const DesktopNavLink: React.FC<Props> = ({ href, label }) => {
  const segment = useSelectedLayoutSegment();
  return (
    <Link
      href={href}
      className={cn("text-white/50 hover:text-white duration-200 text-sm tracking-[0.07px]", {
        "text-white": href.startsWith(`/${segment}`),
      })}
    >
      {label}
    </Link>
  );
};

export const MobileNavLink: React.FC<Props> = ({ href, label }) => {
  const segment = useSelectedLayoutSegment();

  return (
    <Link
      href={href}
      className={cn(
        "text-white/50 hover:text-white duration-200 text-lg font-medium tracking-[0.07px] py-3",
        {
          "text-white": href.startsWith(`/${segment}`),
        },
      )}
    >
      {label}
    </Link>
  );
};
