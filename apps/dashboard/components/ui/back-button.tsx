"use client";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";

function BackButton({
  className,
  children,
}: React.PropsWithChildren<{
  className?: string;
}>) {
  const router = useRouter();
  return (
    <Button type="submit" variant="default" className={className} onClick={() => router.back()}>
      {children}
    </Button>
  );
}

export default BackButton;
