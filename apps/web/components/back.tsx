"use client";
import { ArrowLeft, Loader2 } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import React, { useTransition } from "react";

type Props = {
  href: string;
  label?: string;
};

export const BackLink: React.FC<Props> = ({ href, label }) => {
  const [isPending, startTransition] = useTransition();
  const router = useRouter();

  return (
    <Link
      href={href}
      className="flex items-center gap-1 text-xs duration-200 text-content-subtle hover:text-foreground"
      onClick={() => {
        startTransition(() => {
          router.push(href);
        });
      }}
    >
      {isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <ArrowLeft className="w-4 h-4" />}{" "}
      {label ?? "Back"}
    </Link>
  );
};
