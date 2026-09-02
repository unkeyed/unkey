import { IconUserOutline18 } from "nucleo-ui-outline-18";
import { useState } from "react";
import { cn } from "@/lib/utils";

type AvatarProps = {
  src: string | null | undefined;
  alt: string;
  className?: string;
};

export function Avatar({ src, alt, className }: AvatarProps) {
  const [hasError, setHasError] = useState(false);

  if (!src || hasError) {
    return (
      <div className="size-5  border rounded-full border-grayA-5 items-center flex justify-center">
        <IconUserOutline18 className="size-4" />
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
