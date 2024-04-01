import { CopyButton } from "@/components/copy-button";
import { MDX } from "@/components/mdx-content";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import Image from "next/image";
import Link from "next/link";

import { Changelog } from "@/.contentlayer/generated";
import { Frame } from "../../components/frame";
type Props = {
  changelog: Changelog;
  className?: string;
};

export async function ChangelogGridItem({ className, changelog }: Props) {
  const baseUrl = process.env.VERCEL_URL
    ? `https://${process.env.VERCEL_URL}`
    : "http://localhost:3000";
  const slugable = changelog._raw.flattenedPath.replace("changelog/", "");
  return (
    <div id={`#${slugable}`} className={cn("w-full", className)}>
      <div className="2xl:pl-36">
        <div className="flex flex-row pb-10 gap-4 ">
          {changelog.tags?.map((tag) => (
            <span
              key={tag}
              className="text-white text-xs bg-white/10 rounded-full px-3 py-[0.15rem]"
            >
              {tag.charAt(0).toUpperCase() + tag.slice(1)}
            </span>
          ))}
        </div>
        <h3 className="font-display text-4xl font-medium blog-heading-gradient ">
          <Link href={`#${slugable}`} scroll={false} replace={true}>
            {changelog.title}
          </Link>
        </h3>
        <p className="pt-12">{format(changelog.date, "MMMM dd, yyyy")}</p>
        <p className="my-8 ">{changelog.description}</p>
      </div>
      {changelog.image && (
        <Frame className="shadow-sm my-14 2xl:ml-24" size="md">
          <Image src={changelog.image.toString()} alt={changelog.title} width={1100} height={860} />
        </Frame>
      )}
      <div className="w-full flex flex-col gap-12 2xl:pl-36 2xl:pr-12 prose-thead:border-none">
        <MDX code={changelog.body.code} />
      </div>
      <div>
        <CopyButton value={`${baseUrl}/changelog#${slugable}`} className="mb-6 mt-12 2xl:pl-36 ">
          <p className="">Copy Link</p>
        </CopyButton>
        <Separator orientation="horizontal" className="mb-12" />
      </div>
    </div>
  );
}
