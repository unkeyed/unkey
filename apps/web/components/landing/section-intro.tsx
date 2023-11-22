import clsx from "clsx";

import { Container } from "@/components/landing/container";
import { FadeIn } from "@/components/landing/fade-in";

export function SectionIntro({
  eyebrow,
  title,
  children,
  smaller = false,
  invert = false,
  ...props
}: {
  eyebrow?: string;
  title: string;
  children?: React.ReactNode;
  smaller?: boolean;
  invert?: boolean;
  [key: string]: any;
}) {
  return (
    <Container {...props}>
      <FadeIn className="max-w-2xl">
        <h2>
          {eyebrow && (
            <>
              <span
                className={clsx(
                  "mb-6 block font-display text-base font-semibold",
                  invert ? "text-white" : "text-gray-950",
                )}
              >
                {eyebrow}
              </span>
              <span className="sr-only"> - </span>
            </>
          )}
          <span
            className={clsx(
              "block font-display tracking-tight [text-wrap:balance]",
              smaller ? "text-2xl font-semibold" : "text-4xl font-medium sm:text-5xl",
              invert ? "text-white" : "text-gray-950",
            )}
          >
            {title}
          </span>
        </h2>
        {children && (
          <div className={clsx("mt-6 text-xl", invert ? "text-gray-300" : "text-gray-600")}>
            {children}
          </div>
        )}
      </FadeIn>
    </Container>
  );
}
