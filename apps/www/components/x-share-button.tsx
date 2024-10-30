import { cn } from "@/lib/utils";
import { X } from "lucide-react";
import Link from "next/link";

export const XShareButton = ({ className, url }: { className?: string; url: string }) => {
  return (
    <Link href={url}>
      <button
        type="button"
        className={cn(
          "relative p-1 text-primary focus:outline-none flex items-center gap-2",
          className,
        )}
      >
        <X />
        Share on X
      </button>
    </Link>
  );
};
