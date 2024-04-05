import { RainbowDarkButton } from "@/components/button";
import { Container } from "@/components/container";
import clsx from "clsx";
import { ArrowRight } from "lucide-react";
import Link from "next/link";

export function PageIntro({
  eyebrow,
  title,
  children,
  centered = true,
}: {
  eyebrow?: string;
  title: string;
  children: React.ReactNode;
  centered?: boolean;
}) {
  return (
    <Container className={clsx(centered && "text-center")}>
      <Link
        href="https://github.com/unkeyed/examples"
        className="text-white/60 hover:text-white/80"
        target="_blank"
        rel="noreferrer"
      >
        <RainbowDarkButton
          label="Submit your template"
          IconRight={ArrowRight}
          className="mx-auto mb-12 hover:shadow-md hover:scale-[1.01] flex-shrink-0  sm:mt-12 md:mt-20 lg:mt-32 bg-black"
        />
      </Link>
      <h1>
        <span className="block font-sans text-base font-semibold text-white font-display">
          {eyebrow}
        </span>
        <span className="sr-only"> - </span>
        <span
          className={clsx(
            "mt-6 block max-w-5xl font-display text-5xl font-medium tracking-tight blog-heading-gradient [text-wrap:balance] max-sm:text-4xl sm:mt-2 sm:text-6xl",
            centered && "mx-auto",
          )}
        >
          {title}
        </span>
      </h1>
      <div className={clsx("max-w-3xl text-xl text-white/60", centered && "mx-auto")}>
        {children}
      </div>
    </Container>
  );
}
