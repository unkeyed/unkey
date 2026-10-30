import { IconUserOutline18 } from "@unkey/icons";
import { cn } from "cn";
import { useState } from "react";

type AvatarProps = {
  src: string | null | undefined;
  alt: string;
  className?: string;
};

export function Avatar({ src, alt, className }: AvatarProps) {
  const [failedSrc, setFailedSrc] = useState<string | null>(null);

  if (!src || failedSrc === src) {
    const labelled = alt !== "";
    return (
      <span
        role={labelled ? "img" : undefined}
        aria-label={labelled ? alt : undefined}
        aria-hidden={labelled ? undefined : true}
        data-sentry-mask={labelled || undefined}
        className={cn("size-5 border rounded-full items-center flex justify-center", className)}
      >
        <IconUserOutline18 className="size-3.5" />
      </span>
    );
  }

  return (
    <img
      src={src}
      alt={alt}
      className={cn("size-5 rounded-full object-cover", className)}
      onError={() => setFailedSrc(src)}
    />
  );
}
