import { cn } from "@/lib/utils";
import Image from "next/image";
import { Frame } from "../frame";

export function BlogImage({
  size,
  className,
  imageUrl,
}: {
  size: "sm" | "md" | "lg";
  className?: string;
  imageUrl: string;
}) {
  return (
    <Frame className={cn("shadow-sm", className)} size={size}>
      <Image src={imageUrl!} width={800} height={600} alt="Hero Image" />
    </Frame>
  );
}
