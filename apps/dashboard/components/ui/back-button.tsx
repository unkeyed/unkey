"use client";
import { useRouter } from "next/navigation";
import { Button } from "./button";

function BackButton({
  className,
  variant,
  children,
}: React.PropsWithChildren<{
  className?: string;
  variant?:
    | "primary"
    | "secondary"
    | "outline"
    | "alert"
    | "disabled"
    | "ghost"
    | "link"
    | null
    | undefined;
}>) {
  const router = useRouter();
  return (
    <Button
      type="submit"
      variant={variant ?? "primary"}
      className={className}
      onClick={() => router.back()}
    >
      {children}
    </Button>
  );
}

export default BackButton;
