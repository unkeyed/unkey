import { cn } from "@/lib/utils";
import { User } from "@unkey/icons";
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
      <div className="size-5  border rounded-full border-grayA-5 items-center flex justify-center">
        <User iconSize="md-medium" />
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
