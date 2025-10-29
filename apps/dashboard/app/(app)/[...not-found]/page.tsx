"use client";
import { Button, Empty } from "@unkey/ui";
import { useRouter } from "next/navigation";

export default function NotFound() {
  const router = useRouter();

  return (
    <Empty>
      <Empty.Title>404 Not Found</Empty.Title>
      <Empty.Description>
        We couldn't find the page that you're looking for!
      </Empty.Description>
      <Empty.Actions>
        <Button
          variant="default"
          onClick={() => {
            router.push("/");
          }}
        >
          Go Back
        </Button>
      </Empty.Actions>
    </Empty>
  );
}
