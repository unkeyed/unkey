import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import Link from "next/link";
type Changelog = {
  frontmatter: {
    title: string;
    date: string;
    description: string;
  };
  slug: string;
};

type ChangelogsType = Changelog[];

type SideListProps = {
  logs?: ChangelogsType;
  className?: string;
};

export function SideList({ logs, className }: SideListProps) {
  return (
    <div className={cn("", className)}>
      <ScrollArea className="h-96">
        {logs?.map((log, _index) => (
          <Link href={`/changelog/#${log.slug}`}>
            <p key={log.slug} className="text-sm text-white text-left mb-6 ">
              {format(new Date(log.frontmatter.date), "MMMM dd, yyyy")}
            </p>
          </Link>
        ))}

        <ScrollBar orientation="vertical" forceMount={true} />
      </ScrollArea>
    </div>
  );
}
