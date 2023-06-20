"use client";

import Link from "next/link";
import { useSelectedLayoutSegments } from "next/navigation";
import { Code, Hash } from "lucide-react";

import { Button } from "@/components/ui/button";

type Props = {
  id: string
  href: string;
  name: string ;
};

export const ApiLink: React.FC<Props> = ({ href, name,id }) => {
  const isActive = id === useSelectedLayoutSegments().at(1);
  return (
    <Link href={href}>
      <Button
        variant={isActive ? "default" : "ghost"}
        size="sm"
        className="justify-start w-full font-normal"
      >
        <Code className="w-4 h-4 mr-2" />
        {name}
      </Button>
    </Link>
  );
};
