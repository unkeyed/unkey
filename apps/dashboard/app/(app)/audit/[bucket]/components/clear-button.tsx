"use client";

import { Button } from "@unkey/ui";
import { Loader2, X } from "lucide-react";
import { useRouter } from "next/navigation";
import { useTransition } from "react";

export const ClearButton = () => {
  const router = useRouter();
  const [isPending, startTransition] = useTransition();

  const handleClear = () => {
    startTransition(() => {
      router.push("/audit");
      router.refresh();
    });
  };

  return (
    <Button
      onClick={handleClear}
      className="flex items-center h-8 gap-2 bg-transparent"
      disabled={isPending}
    >
      {isPending ? (
        <>
          Clearing
          <Loader2 className="w-4 h-4 animate-spin" />
        </>
      ) : (
        <>
          Clear
          <X className="w-4 h-4" />
        </>
      )}
    </Button>
  );
};
