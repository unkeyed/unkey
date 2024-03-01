import { cn } from "@/lib/utils";

type SectionTitleProps = {
  label?: string;
  title?: React.ReactNode;
  text?: React.ReactNode;
  align?: "left" | "center";
  children?: React.ReactNode;
  titleWidth?: number;
  contentWidth?: number;
  className?: string;
};

export function SectionTitle({
  label,
  title,
  text,
  align = "left",
  children,
  titleWidth,
  contentWidth,
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
      <p
        className={cn("font-mono text-sm md:text-md text-white/50 text-center", {
          "xl:text-left": align === "left",
        })}
      >
        {label}
      </p>
      <h1
        className={cn(
          "text-[28px] xs:text-[42px] xs-leading[48px] xs:pb-3 sm:text-[52px] sm:leading-[64px] text-white md:max-w-[463px] pt-4 font-medium section-title-heading-gradient text-center",
          { "xl:text-left": align === "left" },
        )}
        style={{ maxWidth: titleWidth ? `${titleWidth}px` : "none" }}
      >
        {title}
      </h1>
      <p
        className={cn("text-sm md:text-base text-white leading-7 py-[26px] text-center", {
          "xl:text-left": align === "left",
        })}
        style={{ maxWidth: contentWidth ? `${contentWidth}px` : "none" }}
      >
        {text}
      </p>
      {children}
    </div>
  );
}
