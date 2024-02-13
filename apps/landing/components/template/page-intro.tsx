import { RainbowDarkButton } from "@/components/button";
import { Container } from "@/components/container";
import clsx from "clsx";
import { ArrowRight } from "lucide-react";

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
    <Container className={clsx("mt-24 sm:mt-32 lg:mt-40", centered && "text-center")}>
      <RainbowDarkButton
        label="Submit Your Template"
        IconRight={ArrowRight}
        className="mx-auto mb-12 hover:shadow-md hover:scale-[1.01]"
      />
      <h1>
        <span className="block font-sans text-base font-semibold font-display text-white">
          {eyebrow}
        </span>
        <span className="sr-only"> - </span>
        <span
          className={clsx(
            "mt-6 block max-w-5xl font-display text-5xl font-medium tracking-tight blog-heading-gradient [text-wrap:balance] sm:text-6xl",
            centered && "mx-auto",
          )}
        >
          {title}
        </span>
      </h1>
      <div className={clsx("mt-6 max-w-3xl text-xl text-white/60", centered && "mx-auto")}>
        {children}
      </div>
    </Container>
  );
}
