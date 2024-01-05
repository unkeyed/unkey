"use client";
import { Button } from "@tremor/react";
import { Plus } from "lucide-react";

import { useState } from "react";

import { toast } from "sonner";
import { createKey } from "./action";

export const NewKeyButton: React.FC = () => {
  const [_key, setKey] = useState<string | undefined>(undefined);
  const [isLoading, setLoading] = useState(false);

  return (
    <div>
      <form
        action={async () => {
          try {
            setLoading(true);
            const key = await createKey();
            setKey(key);
            toast("Copied key to clipboard.");
            navigator.clipboard.writeText(key as string);
          } finally {
            setLoading(false);
          }
        }}
      >
        <Button
          icon={Plus}
          loading={isLoading}
          disabled={isLoading}
          variant="secondary"
          type="submit"
        >
          Create new key
        </Button>
      </form>
    </div>
  );
};
