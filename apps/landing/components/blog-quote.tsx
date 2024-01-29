import { cn } from "@/lib/utils";

type BlogQuoteProps = {
  children?: React.ReactNode;
  className?: string;
  quote?: string;
  author?: string;
};

export function BlogQuote({ children, className, ...props }: BlogQuoteProps) {
  return (
    <div className={cn("flex flex-col text-lg border-l-2 border-white/20 my-12 ml-8 py-4")}>
      <div className="align-middle h-fit p-0 m-0">
        <blockquote className="my-auto pl-12 font-medium text-white leading-8 pr-16">
          {children}
        </blockquote>
        <p className="font-normal text-white/40 leading-8">{props.author}</p>
      </div>
    </div>
  );
}
// export function BlogQuoteText({ children, className }: BlogQuoteProps) {
//   return (
//     <p
//       className={cn(
//         "font-medium text-red-500",

//         className,
//       )}
//     >
//       {children}
//     </p>
//   );
// }
// export function BlogQuoteAuthor({ children, className }: BlogQuoteProps) {
//   return (
//     <p
//       className={cn(
//         "font-normal leading-8 text-white/40",

//         className,
//       )}
//     >
//       <span>— </span>
//       {children}
//     </p>
//   );
// }
