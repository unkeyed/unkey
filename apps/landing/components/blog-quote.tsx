import { cn } from "@/lib/utils";

type BlogQuoteProps = {
  children?: React.ReactNode;
  className?: string;
  author?: string;
};

export function BlogQuote({ children, className, ...props }: BlogQuoteProps) {
  return (
    <div className={cn("flex flex-col text-lg border-l-2 border-white/20 my-12 ml-8 py-4")}>
      <div className="align-middle h-fit p-0 m-0">
        <blockquote className="my-auto pl-12 font-medium text-white leading-8 pr-16">
          {children}
        </blockquote>
        {props.author && <p className="font-normal text-white/40 leading-8">{props.author}</p>}
      </div>
    </div>
  );
}
