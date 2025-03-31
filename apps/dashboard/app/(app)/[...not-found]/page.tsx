import { Empty } from "@unkey/ui";
import { Button } from "@unkey/ui";
import Link from "next/link";
export default function NotFound() {
  return (
    <Empty>
      <Empty.Title>404 Not Found</Empty.Title>
      <Empty.Description>We couldn't find the page that you're looking for!</Empty.Description>
      <Empty.Actions>
        <Link href="/">
          <Button variant="default">Go Back</Button>
        </Link>
      </Empty.Actions>
    </Empty>
  );
}
