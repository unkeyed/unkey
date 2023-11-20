import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Button } from "@/components/ui/button";
import Link from "next/link";
export default function NotFound() {
  return (
    <EmptyPlaceholder className="my-4 ">
      <EmptyPlaceholder.Title>404 Not Found</EmptyPlaceholder.Title>
      <EmptyPlaceholder.Description>
        We couldn't find the page that you're looking for!
      </EmptyPlaceholder.Description>
      <div className="flex flex-col items-center justify-center gap-2 md:flex-row">
        <Link href="/app">
          <Button variant="secondary" className="w-full items-center gap-2 ">
            Go Back
          </Button>
        </Link>
      </div>
    </EmptyPlaceholder>
  );
}
