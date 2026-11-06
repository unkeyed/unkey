import { Button, FormSelect } from "@unkey/ui";
import { useState } from "react";

const regions = [
  { value: "us-east-1", label: "us-east-1" },
  { value: "eu-central-1", label: "eu-central-1" },
];

export default function FormSelectLabelAction() {
  const [region, setRegion] = useState("us-east-1");
  return (
    <div className="w-80">
      <FormSelect
        label="Fallback region"
        labelAction={
          <Button type="button" variant="ghost" size="sm">
            Remove
          </Button>
        }
        size="sm"
        options={regions}
        value={region}
        onValueChange={setRegion}
      />
    </div>
  );
}
