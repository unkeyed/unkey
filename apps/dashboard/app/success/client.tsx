"use client";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export function SuccessClient() {
  const router = useRouter();

  useEffect(() => {
    router.push("/settings/billing");
  }, [router]);

  return <></>;
}
