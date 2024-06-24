import { cn } from "@/lib/utils";
import Image from "next/image";
import { Frame } from "../frame";

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
    <Image
      placeholder="blur"
      blurDataURL="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8+e1bKQAJMQNc5W2CQwAAAABJRU5ErkJggg=="
      src={imageUrl.src!}
      width={1920}
      height={1080}
      alt=""
      className="rounded-md"
    />
  );
}
