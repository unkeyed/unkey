"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";
import React from "react";
import { Button } from "../ui/button";

type Variant =
  | "primary"
  | "secondary"
  | "outline"
  | "alert"
  | "disabled"
  | "ghost"
  | "link"
  | null
  | undefined;

export const CreateKeyButton = (props: { keyAuthId: string; variant?: Variant }) => {
  // Add missing import

  const href = `/keys/${props.keyAuthId}/new`;
  const path = usePathname();
  const setUrl = () => {
    window.location.href = href;
  };
  if (path?.match(href)) {
    return (
      <Button onClick={setUrl} variant={props.variant ?? "primary"}>
        Create Key
      </Button>
    );
  }
  return (
    <Link key="new" href={href}>
      <Button variant={props.variant ?? "primary"}>Create Key</Button>
    </Link>
  );
};
