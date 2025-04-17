"use client";
import { Button } from "@unkey/ui";
import Link from "next/link";
import { usePathname } from "next/navigation";

export const CreateKeyButton = (props: {
  apiId: string;
  keyAuthId: string;
}) => {
  // Add missing import

  const href = `/apis/${props.apiId}/keys/${props.keyAuthId}/new`;
  const path = usePathname();
  const setUrl = () => {
    window.location.href = href;
  };
  if (path?.match(href)) {
    return (
      <Button onClick={setUrl} variant="primary">
        Create Key
      </Button>
    );
  }
  return (
    <Link key="new" href={href}>
      <Button variant="primary">Create Key</Button>
    </Link>
  );
};
