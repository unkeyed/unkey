import { cn } from "@/lib/utils";

type BlogListProps = {
  children?: React.ReactNode;
  className?: string;
};

export function BlogList({ children, className }: BlogListProps) {
  return <ul className={cn("flex flex-col pl-4", className)}>{children}</ul>;
}
export function BlogListItem({ children, className }: BlogListProps) {
  return (
    <li
      className={cn(
        "list-disc pl-6 text-lg leading-8 font-normal text-white/60",

        className,
      )}
    >
      {children}
    </li>
  );
}
