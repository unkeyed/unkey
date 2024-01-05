"use client";
import { Button } from "@tremor/react";
import { Trash } from "lucide-react";
import { useState } from "react";

import { revokeKey } from "./action";
type Props = {
  keyId: string;
};
export const RevokeKeyButton: React.FC<Props> = ({ keyId }) => {
  const [isLoading, setLoading] = useState(false);

  return (
    <div>
      <form
        action={async (formData) => {
          try {
            setLoading(true);
            const _key = await revokeKey(formData);
          } finally {
            setLoading(false);
          }
        }}
      >
        <input id="keyId" name="keyId" value={keyId} readOnly hidden />
        <Button
          icon={Trash}
          loading={isLoading}
          disabled={isLoading}
          variant="secondary"
          size="sm"
          type="submit"
        >
          Revoke
        </Button>
      </form>
    </div>
  );
};
