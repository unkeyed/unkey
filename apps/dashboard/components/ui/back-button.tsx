"use client";
import { useRouter } from "next/navigation";
import { Button } from "@unkey/ui";

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
