"use client";

import { Plus, Trash } from "@unkey/icons";
import { Button, FormInput } from "@unkey/ui";
import type { BasicAuthConfig } from "../types";

export function BasicAuthForm({
  config,
  onChange,
}: {
  config: BasicAuthConfig;
  onChange: (config: BasicAuthConfig) => void;
}) {
  const add = () => {
    onChange({
      ...config,
      credentials: [...config.credentials, { username: "", passwordHash: "" }],
    });
  };

  const update = (index: number, field: "username" | "passwordHash", value: string) => {
    const creds = [...config.credentials];
    creds[index] = { ...creds[index], [field]: value };
    onChange({ ...config, credentials: creds });
  };

  const remove = (index: number) => {
    onChange({ ...config, credentials: config.credentials.filter((_, i) => i !== index) });
  };

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium text-gray-11">Credentials</span>
        <Button variant="ghost" size="sm" onClick={add}>
          <Plus className="size-3" />
          Add
        </Button>
      </div>

      {config.credentials.length === 0 && (
        <p className="text-xs text-grayA-8">No credentials configured.</p>
      )}

      {config.credentials.map((cred, i) => (
        <div
          key={`${cred.username}-${i}`}
          className="flex items-start gap-2 rounded-lg border border-grayA-3 bg-gray-1 p-3"
        >
          <div className="flex-1 flex flex-col gap-2">
            <FormInput
              label="Username"
              value={cred.username}
              onChange={(e) => update(i, "username", e.target.value)}
            />
            <FormInput
              label="Password Hash (BCrypt)"
              value={cred.passwordHash}
              placeholder="$2b$..."
              onChange={(e) => update(i, "passwordHash", e.target.value)}
            />
          </div>
          <Button
            variant="ghost"
            size="sm"
            className="self-start mt-5 text-grayA-8 hover:text-red-10 shrink-0"
            onClick={() => remove(i)}
          >
            <Trash className="size-3" />
          </Button>
        </div>
      ))}
    </div>
  );
}
