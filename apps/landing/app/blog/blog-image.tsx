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
  imageUrl: string;
}) {
  return (
    <Frame className={cn("shadow-sm", className)} size={size}>
      <Image src={imageUrl!} width={1200} height={800} alt="" />
    </Frame>
  );
}
