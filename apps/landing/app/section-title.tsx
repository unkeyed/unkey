import { cn } from "@/lib/utils";

type SectionTitleProps = {
  label?: string;
  title?: string;
  text?: string;
  align?: "left" | "center";
  children?: React.ReactNode;
  className?: string;
  titleWidth?: string;
  contentWidth?: string;
};

export function SectionTitle({
  label,
  title,
  text,
  align = "left",
  children,
  className,
  titleWidth,
  contentWidth,
}: SectionTitleProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center max-w-5xl mx-auto text-center",
        {
          "md:items-start md:text-left": align === "left",
        },
        className,
      )}
    >
      <p className="font-mono text-sm md:text-md text-white/50">{label}</p>
      <h1
        className={cn(
          "text-[28px] md:text-[4rem] leading-9 md:leading-[4rem] text-white text-balance   pt-4 font-medium section-title-heading-gradient",
        )}
      >
        {title}
      </h1>
      <p className="text-sm md:text-base text-white leading-7 py-[26px] max-w-5xl text-balance">
        {text}
      </p>
      {children}
    </div>
  );
}
