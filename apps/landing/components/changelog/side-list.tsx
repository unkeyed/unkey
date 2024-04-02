import type { Changelog } from "@/.contentlayer/generated";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import { format } from "date-fns";

type SideListProps = {
  logs?: Changelog[];
  className?: string;
};

export function SideList({ logs, className }: SideListProps) {
  return (
    <div className={cn("", className)}>
      <ScrollArea className="h-96 changelog-gradient xl:sticky">
        {logs?.map((changelog, _index) => {
          return (
            <a
              key={changelog.tableOfContents.slug}
              href={`#${changelog.tableOfContents.slug}`}
              // scroll={false}
            >
              <p className="text-sm text-white text-left mb-6 ">
                {format(changelog.date, "MMMM dd, yyyy")}
              </p>
            </a>
          );
        })}
        <ScrollBar orientation="vertical" forceMount={true} />
      </ScrollArea>
    </div>
  );
}
