import { cn } from "@/lib/utils";
import { IconUserOutline18 } from "@unkey/icons";
import { useState } from "react";

type AvatarProps = {
  src: string | null | undefined;
  alt: string;
  className?: string;
};

export function Avatar({ src, alt, className }: AvatarProps) {
  const [hasError, setHasError] = useState(false);

  if (!src || hasError) {
    return (
      <div className="size-5  border rounded-full items-center flex justify-center">
        <IconUserOutline18 className="size-3.5" />
      </div>
    );
  }

  return (
    <img
      src={src}
      alt={alt}
      className={cn("size-5 rounded-full object-cover", className)}
      onError={() => setHasError(true)}
    />
  );
}
