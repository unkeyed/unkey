"use client";

import { getErrorMessage } from "@/lib/unkey-client";
import { toast } from "@unkey/ui";

type ToastMessage = { message: string; description?: string };

class StillSettlingError extends Error {}

// Keeps the toast loading until the action's result shows, so success never
// arrives before the page can show it
export function toastWhenSettled(
  settled: Promise<boolean>,
  {
    loading,
    success,
    failure,
  }: { loading: string; success: ToastMessage; failure?: (error: unknown) => ToastMessage },
): void {
  toast.promise(
    settled.then((reached) => {
      if (!reached) {
        throw new StillSettlingError();
      }
    }),
    {
      loading,
      success: () => success,
      error: (error) =>
        error instanceof StillSettlingError
          ? {
              message: "Not finished yet",
              description: "This is taking longer than usual. The page updates when it lands.",
            }
          : (failure?.(error) ?? {
              message: "Something went wrong",
              description: getErrorMessage(error),
            }),
    },
  );
}
