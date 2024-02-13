"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Button } from "../ui/button";

export const CreateKeyButton = (props: { keyAuthId: string }) => {
  const href = `/app/apis/${props.keyAuthId}/new`;
  const path = usePathname();
  const setUrl = () => {
    window.location.href = href;
  };
  if (path?.match(href)) {
    return (
      <Button onClick={setUrl} variant="secondary">
        Create Key
      </Button>
    );
  }
  return (
    <Link key="new" href={href}>
      <Button variant="secondary">Create Key</Button>
    </Link>
  );
};
