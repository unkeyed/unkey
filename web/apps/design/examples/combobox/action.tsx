import { Github, IconPlusOutline12 } from "@unkey/icons";
import { Combobox } from "@unkey/ui";
import { useState } from "react";

const accounts = [
  { value: "acme-inc", label: "acme-inc" },
  { value: "acme-labs", label: "acme-labs" },
];

export default function ComboboxAction() {
  const [account, setAccount] = useState("acme-inc");
  const [added, setAdded] = useState(0);
  return (
    <div className="flex w-56 flex-col gap-2">
      <Combobox
        aria-label="GitHub account"
        leftIcon={<Github />}
        options={accounts}
        value={account}
        onSelect={setAccount}
        searchPlaceholder="Filter accounts…"
        emptyMessage="No accounts match."
        action={{
          label: "Add GitHub account",
          icon: <IconPlusOutline12 />,
          onSelect: () => setAdded((n) => n + 1),
        }}
      />
      <p className="text-gray-10 text-xs">Action ran {added} times</p>
    </div>
  );
}
