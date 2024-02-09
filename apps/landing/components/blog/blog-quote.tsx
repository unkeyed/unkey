import { cn } from "@/lib/utils";
import { Minus } from "lucide-react";
type BlogQuoteProps = {
  children?: React.ReactNode;
  className?: string;
  author?: string;
};

export function BlogQuote({ children, className, ...props }: BlogQuoteProps) {
  return (
    <div
      className={cn(
        "flex flex-col text-lg border-l-2 border-white/20 pl-16 py-4 text-left",
        className,
      )}
    >
      <div className="align-middle h-fit">
        <blockquote className="not-prose my-auto font-medium text-white leading-8">
          {children}
        </blockquote>
        {props.author && (
          <div className="font-normal text-white/40 leading-8 flex flex-row">
            {" "}
            <Minus className="mt-1.5 pr-1" size={20} /> <p className="">{props.author}</p>
          </div>
        )}
      </div>
    </div>
  );
}
