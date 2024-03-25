import { cn } from "@/lib/utils";
import Image from "next/image";
import { Frame } from "../../components/frame";

export function BlogImage({
  size,
  className,
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
    <Frame className={cn("shadow-sm", className)} size={size}>
      <Image src={imageUrl.src!} width={1920} height={1080} alt="" />
    </Frame>
  );
}
