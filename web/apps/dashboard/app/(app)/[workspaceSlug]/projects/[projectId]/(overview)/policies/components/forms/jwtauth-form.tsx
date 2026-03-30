"use client";

import { Switch } from "@/components/ui/switch";
import { FormInput } from "@unkey/ui";
import { SimpleSelect } from "../simple-select";
import type { JWTAuthConfig } from "../types";

export function JWTAuthForm({
  config,
  onChange,
}: {
  config: JWTAuthConfig;
  onChange: (config: JWTAuthConfig) => void;
}) {
  return (
    <div className="flex flex-col gap-4">
      <SimpleSelect
        label="JWKS Source"
        value={config.jwksSource}
        options={[
          { value: "oidcIssuer", label: "OIDC Issuer (auto-discovery)" },
          { value: "jwksUri", label: "JWKS URI" },
          { value: "publicKeyPem", label: "Public Key (PEM)" },
        ]}
        onChange={(v) => onChange({ ...config, jwksSource: v as JWTAuthConfig["jwksSource"] })}
      />

      <FormInput
        label={
          config.jwksSource === "oidcIssuer"
            ? "Issuer URL"
            : config.jwksSource === "jwksUri"
              ? "JWKS URI"
              : "Public Key (PEM)"
        }
        value={config.jwksValue}
        placeholder={
          config.jwksSource === "oidcIssuer"
            ? "https://acme.com"
            : config.jwksSource === "jwksUri"
              ? "https://acme.com/.well-known/jwks.json"
              : "-----BEGIN PUBLIC KEY-----"
        }
        onChange={(e) => onChange({ ...config, jwksValue: e.target.value })}
      />

      <FormInput
        label="Issuer (iss claim)"
        value={config.issuer}
        placeholder="https://acme.com"
        onChange={(e) => onChange({ ...config, issuer: e.target.value })}
      />

      <FormInput
        label="Audiences"
        value={config.audiences.join(", ")}
        placeholder="api.acme.com, admin.acme.com"
        description="Comma-separated audience values"
        onChange={(e) =>
          onChange({
            ...config,
            audiences: e.target.value
              .split(",")
              .map((s) => s.trim())
              .filter(Boolean),
          })
        }
      />

      <FormInput
        label="Algorithms"
        value={config.algorithms.join(", ")}
        placeholder="RS256, ES256"
        description="Comma-separated. Defaults to RS256 if empty."
        onChange={(e) =>
          onChange({
            ...config,
            algorithms: e.target.value
              .split(",")
              .map((s) => s.trim())
              .filter(Boolean),
          })
        }
      />

      <FormInput
        label="Subject Claim"
        value={config.subjectClaim}
        placeholder="sub"
        onChange={(e) => onChange({ ...config, subjectClaim: e.target.value })}
      />

      <FormInput
        label="Forward Claims"
        value={config.forwardClaims.join(", ")}
        placeholder="org_id, plan"
        description="Comma-separated claims to forward to downstream policies"
        onChange={(e) =>
          onChange({
            ...config,
            forwardClaims: e.target.value
              .split(",")
              .map((s) => s.trim())
              .filter(Boolean),
          })
        }
      />

      <div className="flex items-center gap-2">
        <Switch
          id="allow-anonymous"
          checked={config.allowAnonymous}
          onCheckedChange={(allowAnonymous) => onChange({ ...config, allowAnonymous })}
        />
        <label htmlFor="allow-anonymous" className="text-sm text-gray-11 cursor-pointer">
          Allow anonymous requests
        </label>
      </div>
    </div>
  );
}
