import { CopyButton } from "@/components/copy-button";
import { Separator } from "@/components/ui/separator";
import { Frontmatter, Tags } from "@/lib/mdx-helper";
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
  const logsPerPage: number = 4;
  //   const allTags = getAllTags(changelogs);
  //   const selectedTag = searchParams?.tag;
  const filteredLogs = changelogs;
  console.log(filteredLogs.length);

  // selectedTag && selectedTag !== ("all" as Tags)
  //   ? changelogs.filter((p) => p.frontmatter.tags?.includes(selectedTag))
  //   : changelogs;

  const page = Number(searchParams?.page ?? 1);
  console.log(searchParams?.page);

  const visibleLogs = filteredLogs.slice(logsPerPage * (page - 1), logsPerPage * page);

  return (
    <div className={cn(className)}>
      <div className="flex flex-row mt-12">
        <div className="flex flex-col w-96">
          <div className="">
            <SideList logs={visibleLogs} />
          </div>
        </div>
        <div className="flex flex-col ml-20">
          {visibleLogs.map((changelog) => (
            <div className="col-span-full sm:flex sm:items-center sm:justify-between sm:gap-x-8 lg:col-span-1 lg:block">
              <div className="mt-1 flex gap-x-4 sm:mt-0 lg:block">
                <div className="col-span-full lg:col-span-2 ">
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
          //   if (selectedTag) {
          //     newParams.set("tag", selectedTag);
          //   }

          // returns this: /changelog?page=${p}&tag=${tag}
          return `/changelog?${newParams.toString()}`;
        }}
      />
    </div>
  );
};
