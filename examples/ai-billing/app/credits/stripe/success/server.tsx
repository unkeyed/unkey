import { Credits } from "@/components/credits";
import { Button, Card } from "@tremor/react";
import Link from "next/link";

export function Server() {
  return (
    <div className="flex justify-center mt-10">
      <Card className="flex flex-col items-center max-w-[400px]">
        <h1 className="text-center">Payment successful!</h1>
        <Credits />
        <Link href="/">
          <Button className="max-w-[200px] mt-2">Click here to return home</Button>
        </Link>
      </Card>
    </div>
  );
}
