"use client";
import { useClerk } from "@clerk/nextjs";
import { useEffect } from "react";
import type { HandleOAuthCallbackParams } from "@clerk/types";
import { Icons } from "@/components/ui/icons";

export const runtime = "edge";

export default function SSOCallback(props: {
  searchParams: HandleOAuthCallbackParams;
}) {
  const { handleRedirectCallback } = useClerk();

  useEffect(() => {
    void handleRedirectCallback(props.searchParams);
  }, [props.searchParams, handleRedirectCallback]);
  return (
    <div className="h-screen flex items-center justify-center ">
      <Icons.spinner className="mr-2 h-16 w-16 animate-spin" />
    </div>
  );
}
