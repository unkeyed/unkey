import { User } from "@unkey/icons";
import { useState } from "react";

type AvatarProps = {
  src: string | null | undefined;
  alt: string;
  className?: string;
};

export function Avatar({ src, alt, className = "size-5" }: AvatarProps) {
  const [hasError, setHasError] = useState(false);

  if (!src || hasError) {
    return (
      <div className="size-5  border rounded-full border-grayA-5 items-center flex justify-center">
        <User size="md-medium" />
      </div>
    );
  }

  return (
    <img
      src={src}
      alt={alt}
      className={`rounded-full ${className}`}
      onError={() => setHasError(true)}
    />
  );
}
