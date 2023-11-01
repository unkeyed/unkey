import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Button } from "@/components/ui/button";
import { headers } from "next/headers";
import Link from "next/link";
export default function NotFound() {
  const headersList = headers();
  const url = headersList.get("referer") ?? "/app";

  return (
    <EmptyPlaceholder className="my-4 ">
      <EmptyPlaceholder.Title>404 Not Found</EmptyPlaceholder.Title>
      <EmptyPlaceholder.Description>
        We couldn't find the page that you're looking for!
      </EmptyPlaceholder.Description>
      <div className="flex flex-col items-center justify-center gap-2 md:flex-row">
        <Link href={url}>
          <Button variant="secondary" className="items-center w-full gap-2 ">
            Go Back
          </Button>
        </Link>
      </div>
    </EmptyPlaceholder>
  );
}
