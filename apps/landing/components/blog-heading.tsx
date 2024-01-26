import { cn } from "@/lib/utils";

type BlogHeadingProps = {
  children?: React.ReactNode;
  className?: string;
};

export function BlogHeading({ children, className }: BlogHeadingProps) {
  return (
    <div className={cn("flex flex-col text-left w-full pl-24 max-sm:pl-8 pt-20", className)}>
      {children}
    </div>
  );
}
export function BlogTitle({ children, className }: BlogHeadingProps) {
  return (
    <h2
      className={cn(
        "text-5xl max-sm:text-3xl sm:text-3xl md:text-6xl font-medium blog-heading-gradient max-w-xl leading-[64px] md:leading-[86px]",
        className,
      )}
    >
      {children}
    </h2>
  );
}

export function BlogSubTitle({ children, className }: BlogHeadingProps) {
  return (
    <p
      className={cn(
        "text-lg max-sm:text-sm text-white/40 leading-7 max-sm:py-6 max-sm:pr-10 md:py-8",
        className,
      )}
    >
      {children}
    </p>
  );
}
