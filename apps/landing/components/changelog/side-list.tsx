import { Changelog } from "@/.contentlayer/generated";
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
    <div className={cn("", className)}>
      <ScrollArea className="h-96 changelog-gradient">
        {logs?.map((log, _index) => {
          const slug = log._raw.flattenedPath.replace("changelog/", "");
          return (
            <Link key={log.date} href={`#${slug}`} scroll={false} replace={true}>
              <p className="text-sm text-white text-left mb-6 ">
                {format(log.date, "MMMM dd, yyyy")}
              </p>
            </Link>
          );
        })}

        <ScrollBar orientation="vertical" forceMount={true} />
      </ScrollArea>
    </div>
  );
}
