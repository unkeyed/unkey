import clsx from "clsx";

import { Container } from "@/components/landing/container";

export function PageIntro({
  eyebrow,
  title,
  children,
  centered = false,
}: {
  eyebrow?: string;
  title: string;
  children: React.ReactNode;
  centered?: boolean;
}) {
  return (
    <Container className={clsx("mt-24 sm:mt-32 lg:mt-40", centered && "text-center")}>
      <h1>
        <span className="block font-sans text-base font-semibold font-display text-gray-950">
          {eyebrow}
        </span>
        <span className="sr-only"> - </span>
        <span
          className={clsx(
            "mt-6 block max-w-5xl font-display text-5xl font-medium tracking-tight text-gray-950 [text-wrap:balance] sm:text-6xl",
            centered && "mx-auto",
          )}
        >
          {title}
        </span>
      </h1>
      <div className={clsx("mt-6 max-w-3xl text-xl text-gray-600", centered && "mx-auto")}>
        {children}
      </div>
    </Container>
  );
}
