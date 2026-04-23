import { cn } from "@/lib/utils";
import type { ReactNode } from "react";
import { DottedLink } from "../../../../../components/dotted-link";

export const Strong = ({ children, className }: { children: ReactNode; className?: string }) => (
  <span className={cn("text-gray-12 font-medium", className)}>{children}</span>
);

export const Sep = () => <span className="text-gray-9 mx-1.5">·</span>;

export const DocsLink = ({
  href,
  children = "Read more",
}: { href: string; children?: ReactNode }) => <DottedLink href={href}>{children}</DottedLink>;
