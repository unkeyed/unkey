import { MDX } from "@/components/mdx-content";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";
import Image from "next/image";
import Link from "next/link";
import { XShareButton } from "../x-share-button";

import type { Changelog } from "@/.contentlayer/generated";
import { Frame } from "../../components/frame";
type Props = {
  changelog: Changelog;
  className?: string;
};

export async function ChangelogGridItem({ className, changelog }: Props) {
  const baseUrl = process.env.VERCEL_URL
    ? `https://${process.env.VERCEL_URL}`
    : "http://localhost:3000";

  return (
    <div id={changelog.tableOfContents.slug} className={cn("w-full", className)}>
      <div className="">
        <div className="flex flex-col sm:flex-row pb-10 gap-4 font-medium">
          {new Date(changelog.date).toLocaleDateString("en-US", {
            year: "numeric",
            month: "long",
            day: "numeric",
          })}
          <div className="flex flex-row gap-x-3">
            {changelog.tags?.map((tag) => (
              <span
                key={tag}
                className="text-white inline-flex text-xs bg-white/10 rounded-full px-3 py-[0.15rem] leading-[1.5]"
              >
                {tag.charAt(0).toUpperCase() + tag.slice(1)}
              </span>
            ))}
          </div>
        </div>
        <h3 className="font-display text-4xl font-medium blog-heading-gradient ">
          <Link href={`#${changelog.tableOfContents.slug}`} scroll={false} replace={true}>
            {changelog.title}
          </Link>
        </h3>
        <p className="my-8 text-lg font-normal">{changelog.description}</p>
      </div>
      {changelog.image && (
        <Frame className="shadow-sm my-14 2xl:ml-24" size="md">
          <Image src={changelog.image.toString()} alt={changelog.title} width={1100} height={860} />
        </Frame>
      )}
      <div
        className={cn(
          "w-full flex flex-col gap-12 prose-thead:border-none",
          "prose-sm md:prose-md prose-strong:text-white/90 prose-code:text-white/80 prose-code:bg-white/10 prose-code:px-2 prose-code:py-1 prose-code:border-white/20 prose-code:rounded-md prose-pre:p-0 prose-pre:m-0 prose-pre:leading-6",
        )}
      >
        <MDX code={changelog.body.code} />
        <XShareButton
          className="my-2"
          url={`https://twitter.com/intent/post?text=${changelog.title}%0a%0a${baseUrl}/changelog#${changelog.tableOfContents.slug}`}
        />
      </div>
      <div>
        <Separator orientation="horizontal" className="mb-12" />
      </div>
    </div>
  );
}
