import { ChevronRight } from "lucide-react";
import Link from "next/link";

type SectionTitleProps = {
  label: string;
  title: string;
  text: string;
  ctaHref: string;
  ctaText: string;
  secondaryCtaText?: string;
  secondaryCtaHref?: string;
  align: "left" | "center";
};

export function SectionTitle({
  label,
  title,
  text,
  ctaHref,
  ctaText,
  secondaryCtaText,
  secondaryCtaHref,
  align = "center",
}: SectionTitleProps) {
  return (
    <div className="md:pr-24 flex flex-col items-center md:items-start">
      <p className="font-mono text-white/50 text-center md:text-left">{label}</p>
      <h1 className="text-[28px] leading-9 md:text-[52px] text-white md:max-w-[463px] pt-4 section-title-heading-gradient text-center md:text-left">
        {title}
      </h1>
      <p className="text-white leading-7 max-w-[461px] pt-[26px] text-center md:text-left">
        {text}
      </p>
      <Link
        href={ctaHref}
        className="shadow-md mt-[50px] font-medium text-sm bg-white inline-flex items-center border border-white px-4 py-2 rounded-lg gap-2 text-black duration-150 hover:text-white hover:bg-black"
      >
        {ctaText} <ChevronRight className="w-4 h-4" />
      </Link>
      {secondaryCtaText && secondaryCtaHref && (
        <Link
          href={secondaryCtaHref}
          className="shadow-md mt-[50px] font-medium text-sm bg-white inline-flex items-center border border-white px-4 py-2 rounded-lg gap-2 text-black duration-150 hover:text-white hover:bg-black"
        >
          {secondaryCtaText} <ChevronRight className="w-4 h-4" />
        </Link>
      )}
    </div>
  );
}
