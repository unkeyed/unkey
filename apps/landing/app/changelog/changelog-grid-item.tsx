import { CopyButton } from "@/components/copy-button";
import { MdxContent } from "@/components/mdx-content";
import { Separator } from "@/components/ui/separator";
import { Frontmatter } from "@/lib/mdx-helper";
import { getChangelog } from "@/lib/mdx-helper";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import Image from "next/image";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Frame } from "../../components/frame";

type Props = {
  changelog: { frontmatter: Frontmatter; slug: string };
  className?: string;
};

export async function ChangelogGridItem({ className, changelog }: Props) {
  const baseUrl = process.env.VERCEL_URL
    ? `https://${process.env.VERCEL_URL}`
    : "http://localhost:3000";
  const tagList = changelog.frontmatter.tags?.toString().split(" ") || [];
  const { serialized } = await getChangelog(changelog.slug);

  if (!serialized) {
    return notFound();
  }

  return (
    <div id={changelog.slug} className={cn("w-full", className)}>
      <div className="2xl:pl-36">
        <div className="flex flex-row pb-10 gap-4 ">
          {tagList.map((tag) => (
            <span
              key={tag}
              className="text-white text-xs bg-white/10 rounded-full px-3 py-[0.15rem]"
            >
              {tag.charAt(0).toUpperCase() + tag.slice(1)}
            </span>
          ))}
        </div>
        <h3 className="font-display text-4xl font-medium blog-heading-gradient ">
          <Link href={`/changelog#${changelog.slug}`}>{changelog.frontmatter.title}</Link>
        </h3>
        <p className="pt-12">{format(new Date(changelog.frontmatter.date), "MMMM dd, yyyy")}</p>
        <p className="my-8 ">{changelog.frontmatter.description}</p>
      </div>
      {changelog.frontmatter.image && (
        <Frame className="shadow-sm my-14 2xl:ml-24" size="md">
          <Image
            src={changelog.frontmatter.image.toString()}
            alt={changelog.frontmatter.title}
            width={1100}
            height={860}
          />
        </Frame>
      )}
      <div className="w-full flex flex-col gap-12 2xl:pl-36 2xl:pr-12 prose-thead:border-none">
        <MdxContent source={serialized} />
      </div>
      <div>
        <CopyButton
          value={`${baseUrl}/changelog#${changelog.slug}`}
          className="mb-6 mt-12 2xl:pl-36 "
        >
          <p className="">Copy Link</p>
        </CopyButton>
        <Separator orientation="horizontal" className="mb-12" />
      </div>
    </div>
  );
}
