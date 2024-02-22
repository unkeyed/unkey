import { MdxContentChangelog } from "@/components/changelog/mdx-content-changelog";
import { CopyButton } from "@/components/copy-button";
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
  const tagList = changelog.frontmatter.tags?.toString().split(" ") || [];
  const { serialized } = await getChangelog(changelog.slug);

  if (!serialized) {
    return notFound();
  }

  return (
    <div id={changelog.slug} className={cn("pr-8 w-full", className)}>
      <div className="flex flex-row gap-4 pb-10 pl-12">
        {tagList.map((tag) => (
          <span key={tag} className="text-white text-xs bg-white/10 rounded-full px-3 py-[0.15rem]">
            {tag.charAt(0).toUpperCase() + tag.slice(1)}
          </span>
        ))}
      </div>
      <h3 className="font-display text-4xl font-medium blog-heading-gradient pl-12">
        <Link href={`/changelog/${changelog.slug}`}>{changelog.frontmatter.title}</Link>
      </h3>
      <p className="pl-12 pt-12">{format(new Date(changelog.frontmatter.date), "MMMM dd, yyyy")}</p>
      <p className="my-8 pl-12">{changelog.frontmatter.description}</p>
      {changelog.frontmatter.image && (
        <Frame className="shadow-sm my-14" size="lg">
          <Image
            src={changelog.frontmatter.image.toString()}
            alt={changelog.frontmatter.title}
            width={1100}
            height={660}
          />
        </Frame>
      )}
      <div className="prose lg:prose-md prose-neutral dark:prose-invert mx-auto max-w-5xl px-12">
        <MdxContentChangelog source={serialized} />
      </div>
      <div>
        <CopyButton
          value={`https://unkey.dev/changelog/${changelog.slug}`}
          className="mb-6 mt-12 mx-12"
        >
          <p className="pl-2">Copy Link</p>
        </CopyButton>
        <Separator orientation="horizontal" className="mb-12" />
      </div>

      {/* <div className="mt-1 flex gap-x-4 sm:mt-0 lg:block">
        
        <div>
          
          <div className="mt-6 mb-6 flex">
            <Link href={`/changelog/${changelog.slug}`}>
              <p className="text-white">Read more</p>
            </Link>
          </div>
          
          
        </div>
      </div> */}
    </div>
  );
}
