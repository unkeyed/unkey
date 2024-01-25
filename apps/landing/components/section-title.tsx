import { cn } from "@/lib/utils";

type SectionTitleProps = {
  label: string;
  title: string;
  text: string;
  align?: "left" | "center";
  children?: React.ReactNode;
  titleWidth?: number;
  contentWidth?: number;
};

export function SectionTitle({
  label,
  title,
  text,
  align = "left",
  children,
  titleWidth,
  contentWidth,
}: SectionTitleProps) {
  return (
    <div
      className={cn("flex flex-col items-center", {
        "md:items-start": align === "left",
      })}
    >
      <p
        className={cn("font-mono text-sm md:text-md text-white/50 text-center", {
          "md:text-left": align === "left",
        })}
      >
        {label}
      </p>
      <h1
        className={cn(
          "text-[28px] md:text-[52px] leading-9 md:leading-[64px] text-white md:max-w-[463px] pt-4 font-medium section-title-heading-gradient text-center",
          { "md:text-left": align === "left" },
        )}
        style={{ maxWidth: titleWidth ? `${titleWidth}px` : "none" }}
      >
        {title}
      </h1>
      <p
        className={cn("text-sm md:text-md text-white leading-7 py-[26px] text-center", {
          "md:text-left": align === "left",
        })}
        style={{ maxWidth: contentWidth ? `${contentWidth}px` : "none" }}
      >
        {text}
      </p>
      {children}
    </div>
  );
}
