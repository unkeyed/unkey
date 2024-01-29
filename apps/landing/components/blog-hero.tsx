"use client";
import { authors } from "@/content/blog/authors";
import Image from "next/image";
import Link from "next/link";
import { Frame } from "./frame";
type BlogHeroProps = {
  label?: string;
  imageUrl?: string;
  title?: string;
  subTitle?: string;
  author: string;
  publishDate?: string;
  children?: React.ReactNode;
  className?: string;
};

export function BlogHero({
  label,
  imageUrl,
  title,
  subTitle,
  author,
  publishDate,
  children,
  className,
}: BlogHeroProps) {
  return (
    <div className="flex flex-col lg:flex-row w-full text-white">
      <Frame>
        <Image src={imageUrl!} width={1920} height={1080} alt="Hero Image" />
      </Frame>
      <div className="w-full p-12">
        <div className="relative top-0 left-0 text-white/50 text-sm bg-white/10 px-[9px] rounded-md w-fit leading-6 ">
          {label}
        </div>
        <h2 className="font-medium text-3xl leading-10 blog-heading-gradient my-6">{title}</h2>
        <p className="text-base leading-6 font-normal text-white/60">{subTitle}</p>
        <div className="flex flex-row w-full mt-10 gap-24">
          <div className="flex flex-col gap-6 text-nowrap">
            <p className="text-white/30 text-sm ">Written by</p>
            <div>
              <Image
                alt={authors[author]?.name}
                src={authors[author]?.image.src}
                width={12}
                height={12}
                className="h-12 w-12 object-cover grayscale"
              />
              {/* <p className="text-white text-sm">{}</p> */}
            </div>
          </div>
          <div className="flex flex-col gap-6">
            <p className="text-white/30 text-sm">Published on</p>
            <div>
              <p className="text-white text-sm">{publishDate!}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
