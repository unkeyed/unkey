import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import Link from "next/link";

type SideListProps = {
  list: { href: string; label: string }[];
  className?: string;
};

export function SideList({ list, className }: SideListProps) {
  return (
    <ScrollArea className={cn("h-96 changelog-gradient", className)}>
      {list?.map((l) => {
        return (
          <Link
            key={l.href}
            href={l.href}
            className="hover:text-white"
            // scroll={false}
          >
            <p className="text-sm text-white/80 hover:text-white duration-200 text-left mb-6 ">
              {l.label}
            </p>
          </Link>
        );
      })}
      <ScrollBar orientation="vertical" forceMount={true} />
    </ScrollArea>
  );
}
