"use client";

import { FormTextarea } from "@unkey/ui";
import type { OpenAPIConfig } from "../types";

export function OpenAPIForm({
  config,
  onChange,
}: {
  config: OpenAPIConfig;
  onChange: (config: OpenAPIConfig) => void;
}) {
  return (
    <FormTextarea
      label="OpenAPI Spec (YAML)"
      value={config.specYaml}
      placeholder={"openapi: '3.0.0'\ninfo:\n  title: My API\n  version: '1.0'"}
      rows={12}
      className="font-mono text-xs"
      onChange={(e) => onChange({ ...config, specYaml: e.target.value })}
      description="Paste your OpenAPI 3.0/3.1 spec here. Requests are validated against it."
    />
  );
}
