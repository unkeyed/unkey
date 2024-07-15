import type { Author } from "@/content/blog/authors";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import { Frame } from "../frame";
import { ImageWithBlur } from "../image-with-blur";
import { Avatar, AvatarFallback, AvatarImage } from "../ui/avatar";

type BlogCardProps = {
  tags?: string[];
  imageUrl?: string;
  title?: string;
  subTitle?: string;
  author: Author;
  publishDate?: string;
  className?: string;
};

export function BlogCard({
  tags,
  imageUrl,
  title,
  subTitle,
  author,
  publishDate,
  className,
}: BlogCardProps) {
  return (
    <div
      className={cn(
        "flex flex-col rounded-3xl max-sm:h-full duration-150 ease-out border-transparent border hover:border-neutral-900 hover:bg-neutral-950 p-3",
        className,
      )}
    >
      <div className="w-full rounded-2xl bg-clip-border">
        <Frame size="sm">
          <div className="relative aspect-video">
            <ImageWithBlur
              placeholder="blur"
              blurDataURL="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8+e1bKQAJMQNc5W2CQwAAAABJRU5ErkJggg=="
              src={imageUrl!}
              alt="Hero Image"
              className="object-center w-full overflow-hidden"
              fill={true}
            />
          </div>
        </Frame>
      </div>
      <div className="flex flex-col h-full px-1">
        <div className="flex flex-col h-80 pt-6 pb-2">
          <div className="flex flex-wrap h-6 gap-4 flex-inline">
            {tags?.map((tag) => (
              <div className="text-white/50 text-sm bg-white/10 px-[9px] rounded-md content-center">
                {tag.charAt(0).toUpperCase() + tag.slice(1)}
              </div>
            ))}
          </div>
          <h2 className="flex justify-start mt-6 text-xl font-medium leading-10 md:text-2xl sm:text-2xl blog-heading-gradient">
            {title}
          </h2>

          <p className="h-full mt-6 text-base font-normal leading-6 sm:text-sm text-white/60">
            {subTitle}
          </p>
          {/* Todo: Needs ability to add multiple authors at some point */}
          <div className="flex flex-col flex-wrap justify-end h-full">
            <div className="flex flex-row">
              <Avatar className="w-8 h-8">
                <AvatarImage alt={author.name} src={author.image.src} width={12} height={12} />
                <AvatarFallback />
              </Avatar>
              <p className="pt-3 ml-4 text-sm font-medium text-white">{author.name}</p>
              <p className="pt-3 ml-6 text-sm font-normal text-white/50">
                {format(new Date(publishDate!), "MMM dd, yyyy")}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
