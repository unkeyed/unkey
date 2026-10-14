import { FormSelect } from "@unkey/ui";
import { useState } from "react";

const fields = [
  { value: "path", label: "Path" },
  { value: "method", label: "Method" },
] as const;

const methods = ["GET", "POST", "PUT", "DELETE"].map((m) => ({ value: m, label: m }));

export default function FormSelectInline() {
  const [field, setField] = useState<(typeof fields)[number]["value"]>("path");
  const [picked, setPicked] = useState<string[]>(["GET"]);
  return (
    <div className="flex w-96 items-center gap-2">
      <FormSelect
        size="sm"
        aria-label="Condition field"
        options={fields}
        value={field}
        onValueChange={setField}
        className="w-32 shrink-0"
      />
      <FormSelect
        multiple
        size="sm"
        aria-label="Methods"
        placeholder="Pick methods"
        options={methods}
        value={picked}
        onValueChange={setPicked}
        className="min-w-0 flex-1"
      />
    </div>
  );
}
