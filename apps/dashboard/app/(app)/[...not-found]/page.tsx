import { Empty } from "@unkey/ui";
import { Button } from "@unkey/ui";
import Link from "next/link";
export default function NotFound() {
  return (
    <Empty className="my-4" fill={true}>
      <Empty.Title>404 Not Found</Empty.Title>
      <Empty.Description>We couldn't find the page that you're looking for!</Empty.Description>
      <div className="flex flex-col items-center justify-center gap-2 md:flex-row">
        <Link href="/">
          <Button variant="default">Go Back</Button>
        </Link>
      </div>
    </Empty>
  );
}
