"use client";

import Link from "next/link";
import { useSelectedLayoutSegments } from "next/navigation";
import { Hash } from "lucide-react";

import { Button } from "@/components/ui/button";

type Props = {
  href: string;
  slug: string | null;
};

export const AppLink: React.FC<Props> = ({ href, slug }) => {
  const isActive = slug === useSelectedLayoutSegments().at(1);
  return (
    <Link href={href}>
      <Button
        variant={isActive ? "default" : "ghost"}
        size="sm"
        className="justify-start w-full font-normal"
      >
        <Hash className="w-4 h-4 mr-2" />
        {slug}
      </Button>
    </Link>
  );
};
