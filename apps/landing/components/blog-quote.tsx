import { cn } from "@/lib/utils";

type BlogQuoteProps = {
  children?: React.ReactNode;
  className?: string;
};

export function BlogQuote({ children, className }: BlogQuoteProps) {
  return (
    <ul className={cn("flex flex-col text-lg border-l-white/20 py-4 pr-16 pl-12", className)}>
      {children}
    </ul>
  );
}
export function BlogQuoteText({ children, className }: BlogQuoteProps) {
  return (
    <p
      className={cn(
        "font-medium text-white",

        className,
      )}
    >
      {children}
    </p>
  );
}
export function BlogQuoteAuthor({ children, className }: BlogQuoteProps) {
  return (
    <p
      className={cn(
        "font-normal leading-8 text-white/40",

        className,
      )}
    >
      <span>— </span>
      {children}
    </p>
  );
}
