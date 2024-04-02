import { OssLight } from "@/components/svg/oss-light";
import { cn } from "@/lib/utils";
import type { PropsWithChildren } from "react";

type SectionTitleProps = {
  label?: string;
  title?: React.ReactNode;
  text?: React.ReactNode;
  align?: "left" | "center";
  children?: React.ReactNode;
  className?: string;
};

export function SectionTitle({
  label,
  title,
  text,
  align = "left",
  children,
  className,
}: SectionTitleProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center",
        {
          "xl:items-start": align === "left",
        },
        className,
      )}
    >
      <span
        className={cn("font-mono text-sm md:text-md text-white/50 text-center", {
          "xl:text-left": align === "left",
        })}
      >
        {label}
      </span>
      <h2
        className={cn(
          "text-[28px] sm:pb-3 sm:text-[52px] sm:leading-[64px] text-white  max-w-sm md:max-w-md lg:max-w-2xl xl:max-w-4xl  pt-4 font-medium section-title-heading-gradient text-center",
          { "xl:text-left": align === "left" },
        )}
      >
        {title}
      </h2>
      <p
        className={cn(
          "text-sm md:text-base text-white leading-7 py-6 text-center max-w-sm md:max-w-md lg:max-w-2xl xl:max-w-4xl ",
          {
            "xl:text-left xl:max-w-xl": align === "left",
          },
        )}
      >
        {text}
      </p>
      {children}
    </div>
  );
}

export const Section: React.FC<PropsWithChildren<{ className?: string }>> = ({
  children,
  className,
}) => {
  return <section className={cn(className)}>{children}</section>;
};
