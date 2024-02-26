// import { CopyButton } from "@/components/copy-button";
// import { Separator } from "@/components/ui/separator";
import { Frontmatter } from "@/lib/mdx-helper";
import { cn } from "@/lib/utils";
import { ChangelogGridItem } from "./changelog-grid-item";
// import Link from "next/link";
// import { BlogPagination } from "../blog/blog-pagination";
import { SideList } from "./side-list";

type Props = {
  changelogs: { frontmatter: Frontmatter; slug: string }[];
  className?: string;
  searchParams?: {
    page?: number;
  };
};

export const ChangelogGrid: React.FC<Props> = ({ className, changelogs, searchParams }) => {
  // const logsPerPage: number = 20;
  // const filteredLogs = changelogs;
  // const page = Number(searchParams?.page ?? 1);
  // const visibleLogs = filteredLogs.slice(logsPerPage * (page - 1), logsPerPage * page);
  return (
    <div className={cn(className)}>
      <div className="flex flex-row mt-20 mb-20">
        <div className="flex flex-col w-96 px-0">
          <SideList logs={changelogs} className="sticky top-0 mt-0 pt-0" />
        </div>
        <div className="flex flex-col px-0 mx-0 w-full">
          {changelogs.map((changelog) => (
            <ChangelogGridItem key={changelog.slug} changelog={changelog} />
          ))}
        </div>
      </div>
      {/* <BlogPagination
        currentPage={page}
        numPages={Math.ceil(filteredLogs.length / logsPerPage)}
        buildPath={(p: number) => {
          const newParams = new URLSearchParams();
          newParams.set("page", p.toString());
          return `/changelog?${newParams.toString()}`;
        }}
      /> */}
    </div>
  );
};
