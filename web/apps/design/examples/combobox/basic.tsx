import { Combobox } from "@unkey/ui";
import { useState } from "react";

const regions = [
  { value: "us-east-1", label: "us-east-1" },
  { value: "eu-central-1", label: "eu-central-1" },
  { value: "ap-southeast-1", label: "ap-southeast-1" },
];

export default function ComboboxBasic() {
  const [region, setRegion] = useState("us-east-1");
  return (
    <div className="w-56">
      <Combobox
        aria-label="Region"
        options={regions}
        value={region}
        onSelect={setRegion}
        searchPlaceholder="Filter regions…"
      />
    </div>
  );
}
