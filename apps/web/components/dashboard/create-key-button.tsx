"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Button } from "../ui/button";

export const CreateKeyButton = (props: { apiId: string }) => {
  const path = usePathname();
  const setUrl = () => {
    window.location.href = `/app/apis/${props.apiId}/keys/new`;
  };
  if (path?.match(`/app/apis/${props.apiId}/keys/new`)) {
    return (
      <Button onClick={setUrl} variant="secondary">
        Create Key
      </Button>
    );
  } else {
    return (
      <Link key="new" href={`/app/apis/${props.apiId}/keys/new`}>
        <Button variant="secondary">Create Key</Button>
      </Link>
    );
  }
};
