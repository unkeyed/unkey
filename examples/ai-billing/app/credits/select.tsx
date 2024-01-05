"use client";
import { Button, Select, SelectItem } from "@tremor/react";
import Link from "next/link";
import { useState } from "react";

export function SelectCredits() {
  const [value, setValue] = useState("");
  return (
    <div>
      <Select className="mt-10" value={value} onValueChange={(v) => setValue(v)}>
        <SelectItem value="1">10</SelectItem>
        <SelectItem value="2">20</SelectItem>
        <SelectItem value="3">30</SelectItem>
      </Select>
      <div className="flex justify-end mt-10">
        <Link href={`/credits/stripe?value=${value}`}>
          <Button disabled={!value.length} className="ml-auto">
            Purchase with Stripe
          </Button>
        </Link>
      </div>
    </div>
  );
}
