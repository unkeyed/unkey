import { ImageWithBlur } from "../image-with-blur";
export type BlogImageProps = JSX.IntrinsicAttributes & {
  
  size: "sm" | "md" | "lg" | undefined;
  className?: string;
  unoptimize?: boolean;
  src: string | undefined;
  alt?: string | undefined;
};
export function BlogImage(props: BlogImageProps) {

  if (!props.src && !props.alt) {
    return null;
  }
  return (
    <ImageWithBlur
      src={props.src ?? ""}
      unoptimized={props.unoptimize}
      sizes={props.size ?? "sm"}
      alt={props.alt || ""}
      className="rounded-md"
    />
  );
}
