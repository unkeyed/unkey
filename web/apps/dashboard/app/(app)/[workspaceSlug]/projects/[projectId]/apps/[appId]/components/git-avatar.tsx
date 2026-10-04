import { IconUserOutline18 } from "@unkey/icons";
import { cn } from "cn";
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
      <div
        data-sensitive-media=""
        className="size-5  border rounded-full items-center flex justify-center"
      >
        <IconUserOutline18 className="size-3.5" />
      </div>
    );
  }

  return (
    <img
      src={src}
      alt={alt}
      data-sensitive-media=""
      className={cn("size-5 rounded-full object-cover", className)}
      onError={() => setHasError(true)}
    />
  );
}
