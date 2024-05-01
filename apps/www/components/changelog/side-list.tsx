import type { Changelog } from "@/.contentlayer/generated";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import Link from "next/link";

type SideListProps = {
  logs?: Changelog[];
  className?: string;
};

export function SideList({ logs, className }: SideListProps) {
  return (
    <ScrollArea className={cn("h-96 changelog-gradient", className)}>
      {logs?.map((changelog, _index) => {
        return (
          <Link
            key={changelog.tableOfContents.slug}
            href={`#${changelog.tableOfContents.slug}`}
            className="hover:text-white"
            // scroll={false}
          >
            <p className="text-sm text-white/80 hover:text-white duration-200 text-left mb-6 ">
              {format(changelog.date, "MMMM dd, yyyy")}
            </p>
          </Link>
        );
      })}
      <ScrollBar orientation="vertical" forceMount={true} />
    </ScrollArea>
  );
}
