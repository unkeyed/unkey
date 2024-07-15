import { ImageWithBlur } from "../image-with-blur";

export function BlogImage({
  imageUrl,
}: {
  size: "sm" | "md" | "lg";
  className?: string;
  imageUrl: {
    src?: string;
    alt?: string;
  };
}) {
  return (
    <ImageWithBlur src={imageUrl.src!} width={1920} height={1080} alt="" className="rounded-md" />
  );
}
