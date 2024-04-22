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
  return <Image src={imageUrl.src!} width={1920} height={1080} alt="" className="rounded-md" />;
}
