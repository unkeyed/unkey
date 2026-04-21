import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

export const Strong = ({ children, className }: { children: ReactNode; className?: string }) => (
  <span className={cn("text-gray-12 font-medium", className)}>{children}</span>
);

export const Sep = () => <span className="text-gray-9 mx-1.5">·</span>;

export const DocsLink = ({
  href,
  children = "Read more",
}: { href: string; children?: ReactNode }) => (
  <a href={href} target="_blank" rel="noopener noreferrer">
    <span className="font-medium text-gray-12 underline underline-offset-2 decoration-grayA-6 hover:decoration-gray-12 transition-colors decoration-dotted">
      {children}
    </span>
  </a>
);
