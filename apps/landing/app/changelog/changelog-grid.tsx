import { CopyButton } from "@/components/copy-button";
import { Separator } from "@/components/ui/separator";
import { Frontmatter } from "@/lib/mdx-helper";
import { cn } from "@/lib/utils";
import Link from "next/link";
import { BlogPagination } from "../blog/blog-pagination";
import { SideList } from "./side-list";

type Props = {
  changelogs: { frontmatter: Frontmatter; slug: string }[];
  className?: string;
  searchParams?: {
    page?: number;
  };
};

export const ChangelogGrid: React.FC<Props> = ({ className, changelogs, searchParams }) => {
  const logsPerPage: number = 3;
  const filteredLogs = changelogs;
  const page = Number(searchParams?.page ?? 1);
  const visibleLogs = filteredLogs.slice(logsPerPage * (page - 1), logsPerPage * page);
  return (
    <div className={cn(className)}>
      <div className="flex flex-row mt-20 mb-20">
        <div className="flex flex-col w-96 px-0">
          <div>
            <SideList logs={changelogs} />
          </div>
        </div>
        <div className="flex flex-col px-0 mx-0 w-full">
          {visibleLogs.map((changelog) => (
            <div className="sm:flex sm:items-center sm:justify-between sm:gap-x-8 lg:block">
              <div className="mt-1 flex gap-x-4 sm:mt-0 lg:block">
                <div>
                  <h3 className="font-display text-4xl font-medium blog-heading-gradient">
                    <Link href={`/changelog/${changelog.slug}`}>{changelog.frontmatter.title}</Link>
                  </h3>
                  <p className="mt-10">{changelog.frontmatter.description}</p>
                  <div className="mt-6 mb-6 flex">
                    <Link href={`/changelog/${changelog.slug}`}>
                      <p className="text-white">Read more</p>
                    </Link>
                  </div>
                  <Separator orientation="horizontal" className="mb-6" />
                  <div>
                    <CopyButton
                      value={`https://unkey.dev/changelog/${changelog.slug}`}
                      className="mb-6"
                    >
                      <p className="pl-2">Copy Link</p>
                    </CopyButton>
                    <Separator orientation="horizontal" className="mb-12" />
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
      <BlogPagination
        currentPage={page}
        numPages={Math.ceil(filteredLogs.length / logsPerPage)}
        buildPath={(p: number) => {
          const newParams = new URLSearchParams();
          newParams.set("page", p.toString());
          return `/changelog?${newParams.toString()}`;
        }}
      />
    </div>
  );
};
