import { ImageWithBlur } from "../image-with-blur";

export function BlogImage({
  imageUrl,
}: {
  size: "sm" | "md" | "lg";
  className?: string;
  unoptimized?: boolean;
  imageUrl: {
    src: string;
    alt?: string;
  };
}) {
  return (
    <ImageWithBlur
      src={imageUrl.src}
      width={1920}
      unoptimized
      height={1080}
      alt={imageUrl.alt || ""}
      className="rounded-md"
    />
  );
}
