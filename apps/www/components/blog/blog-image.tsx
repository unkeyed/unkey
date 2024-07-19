import { ImageWithBlur } from "../image-with-blur";

export function BlogImage({
  imageUrl,
  unoptimize,
  size,
}: {
  size: "sm" | "md" | "lg";
  className?: string;
  unoptimize?: boolean;
  imageUrl: {
    src: string;
    alt?: string;
  };
}) {
  return (
    <ImageWithBlur
      src={imageUrl.src}
      width={1920}
      unoptimized={unoptimize}
      height={1080}
      sizes={size}
      alt={imageUrl.alt || ""}
      className="rounded-md"
    />
  );
}
