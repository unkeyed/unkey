"use client";

import { trpc } from "@/lib/trpc/client";
import { Clock, Link4 } from "@unkey/icons";
import { CopyButton, toast } from "@unkey/ui";
import { useState } from "react";

// Creates a one-time share link for the just-created secret. The secret is
// vault-encrypted server-side; the link carries only an opaque id.
export const ShareKeyButton = ({ secret }: { secret: string }) => {
  const [link, setLink] = useState<string | null>(null);

  const create = trpc.share.create.useMutation({
    onSuccess: ({ id }) => {
      // The id rides in the fragment (after `#`) so it stays out of server
      // access logs and Referer headers.
      setLink(`${window.location.origin}/share#${id}`);
    },
    onError: (err) => {
      console.error(err);
      toast.error("Could not create a secure link. Please try again.");
    },
  });

  const isGenerating = create.isLoading;
  const generate = () => create.mutate({ secret });

  if (link) {
    return (
      <div className="w-full mt-6 flex flex-col gap-2 items-start">
        <div className="w-full px-4 py-2 bg-white dark:bg-black border rounded-xl border-grayA-5">
          <div className="flex items-center justify-between w-full gap-3">
            <Link4 iconSize="sm-regular" className="text-gray-12 shrink-0" />
            <p className="flex-1 overflow-x-auto min-w-0 whitespace-pre-wrap break-all font-mono text-[13px] text-grayA-12">
              {link}
            </p>
            <div className="shrink-0">
              <CopyButton value={link} title="Copy secure link" />
            </div>
          </div>
        </div>
        <div className="text-gray-9 text-[13px] flex items-center gap-1.5 self-center">
          <Clock className="text-warning-9" iconSize="sm-regular" />
          <span>
            Reveals the key once, expires in 72h.{" "}
            <button
              type="button"
              onClick={generate}
              disabled={isGenerating}
              className="text-info-11 hover:underline disabled:opacity-50 disabled:no-underline"
            >
              {isGenerating ? "Regenerating…" : "Regenerate"}
            </button>
          </span>
        </div>
      </div>
    );
  }

  return (
    <div className="w-full mt-4 text-center">
      <button
        type="button"
        onClick={generate}
        disabled={isGenerating}
        className="text-info-11 hover:underline text-[13px] disabled:opacity-50 disabled:no-underline"
      >
        {isGenerating ? "Creating link…" : "Share as a secure link"}
      </button>
    </div>
  );
};
