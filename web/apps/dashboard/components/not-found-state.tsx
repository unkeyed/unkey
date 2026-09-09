"use client";
import { Button, Empty } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { routes } from "@/lib/navigation/routes";

type NotFoundStateProps = {
  title?: string;
  description?: string;
};

export function NotFoundState({
  title = "404 Not Found",
  description = "We couldn't find the page that you're looking for!",
}: NotFoundStateProps) {
  const router = useRouter();

  return (
    <Empty>
      <Empty.Title>{title}</Empty.Title>
      <Empty.Description>{description}</Empty.Description>
      <Empty.Actions>
        <Button
          variant="default"
          onClick={() => {
            router.push(routes.workspaces.root());
          }}
        >
          Go Back
        </Button>
      </Empty.Actions>
    </Empty>
  );
}
