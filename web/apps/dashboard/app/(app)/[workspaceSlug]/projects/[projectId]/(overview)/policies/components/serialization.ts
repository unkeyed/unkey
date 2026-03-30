import { BasicAuthCredentialSchema, BasicAuthSchema } from "@/gen/proto/policies/v1/basicauth_pb";
import { IPRulesSchema } from "@/gen/proto/policies/v1/iprules_pb";
import { JWTAuthSchema } from "@/gen/proto/policies/v1/jwtauth_pb";
import {
  BearerTokenLocationSchema,
  HeaderKeyLocationSchema,
  KeyAuthSchema,
  KeyLocationSchema,
  QueryParamKeyLocationSchema,
} from "@/gen/proto/policies/v1/keyauth_pb";
import {
  HeaderMatchSchema,
  MatchExprSchema,
  MethodMatchSchema,
  PathMatchSchema,
  QueryParamMatchSchema,
  StringMatchSchema,
} from "@/gen/proto/policies/v1/match_pb";
import { OpenApiRequestValidationSchema } from "@/gen/proto/policies/v1/openapi_pb";
import { PolicySchema } from "@/gen/proto/policies/v1/policy_pb";
import type { Policy } from "@/gen/proto/policies/v1/policy_pb";
import {
  AuthenticatedSubjectKeySchema,
  HeaderKeySchema,
  PathKeySchema,
  PrincipalClaimKeySchema,
  RateLimitKeySchema,
  RateLimitSchema,
  RemoteIpKeySchema,
} from "@/gen/proto/policies/v1/ratelimit_pb";
import { create } from "@bufbuild/protobuf";
import type {
  BasicAuthConfig,
  IPRulesConfig,
  JWTAuthConfig,
  KeyAuthConfig,
  MatchFormData,
  OpenAPIConfig,
  PolicyFormData,
  PolicyType,
  RateLimitConfig,
  StringMatchMode,
} from "./types";

// ── Form data → Protobuf ──────────────────────────────────────────────

export function toProtobufPolicy(form: PolicyFormData): Policy {
  const policy = create(PolicySchema, {
    id: form.id,
    name: form.name,
    enabled: form.enabled,
    match: form.match.map(toProtobufMatch),
  });

  switch (form.type) {
    case "keyauth": {
      const c = form.config as KeyAuthConfig;
      policy.config = {
        case: "keyauth",
        value: create(KeyAuthSchema, {
          keySpaceIds: c.keySpaceIds,
          locations: c.locations.map((loc) => {
            switch (loc.type) {
              case "bearer":
                return create(KeyLocationSchema, {
                  location: { case: "bearer", value: create(BearerTokenLocationSchema) },
                });
              case "header":
                return create(KeyLocationSchema, {
                  location: {
                    case: "header",
                    value: create(HeaderKeyLocationSchema, {
                      name: loc.name,
                      stripPrefix: loc.stripPrefix ?? "",
                    }),
                  },
                });
              case "queryParam":
                return create(KeyLocationSchema, {
                  location: {
                    case: "queryParam",
                    value: create(QueryParamKeyLocationSchema, { name: loc.name }),
                  },
                });
            }
          }),
          permissionQuery: c.permissionQuery || undefined,
        }),
      };
      break;
    }
    case "jwtauth": {
      const c = form.config as JWTAuthConfig;
      const jwksSource = (() => {
        switch (c.jwksSource) {
          case "jwksUri":
            return { case: "jwksUri" as const, value: c.jwksValue };
          case "oidcIssuer":
            return { case: "oidcIssuer" as const, value: c.jwksValue };
          case "publicKeyPem":
            return { case: "publicKeyPem" as const, value: new TextEncoder().encode(c.jwksValue) };
        }
      })();
      policy.config = {
        case: "jwtauth",
        value: create(JWTAuthSchema, {
          jwksSource,
          issuer: c.issuer,
          audiences: c.audiences,
          algorithms: c.algorithms,
          subjectClaim: c.subjectClaim,
          forwardClaims: c.forwardClaims,
          allowAnonymous: c.allowAnonymous,
          clockSkewMs: BigInt(0),
          jwksCacheMs: BigInt(0),
        }),
      };
      break;
    }
    case "basicauth": {
      const c = form.config as BasicAuthConfig;
      policy.config = {
        case: "basicauth",
        value: create(BasicAuthSchema, {
          credentials: c.credentials.map((cred) =>
            create(BasicAuthCredentialSchema, {
              username: cred.username,
              passwordHash: cred.passwordHash,
            }),
          ),
        }),
      };
      break;
    }
    case "ratelimit": {
      const c = form.config as RateLimitConfig;
      const key = create(RateLimitKeySchema, {
        source: (() => {
          switch (c.keySource) {
            case "remoteIp":
              return { case: "remoteIp" as const, value: create(RemoteIpKeySchema) };
            case "header":
              return {
                case: "header" as const,
                value: create(HeaderKeySchema, { name: c.keyValue }),
              };
            case "authenticatedSubject":
              return {
                case: "authenticatedSubject" as const,
                value: create(AuthenticatedSubjectKeySchema),
              };
            case "path":
              return { case: "path" as const, value: create(PathKeySchema) };
            case "principalClaim":
              return {
                case: "principalClaim" as const,
                value: create(PrincipalClaimKeySchema, { claimName: c.keyValue }),
              };
          }
        })(),
      });
      policy.config = {
        case: "ratelimit",
        value: create(RateLimitSchema, {
          limit: BigInt(c.limit),
          windowMs: BigInt(c.windowMs),
          key,
        }),
      };
      break;
    }
    case "ipRules": {
      const c = form.config as IPRulesConfig;
      policy.config = {
        case: "ipRules",
        value: create(IPRulesSchema, { allow: c.allow, deny: c.deny }),
      };
      break;
    }
    case "openapi": {
      const c = form.config as OpenAPIConfig;
      policy.config = {
        case: "openapi",
        value: create(OpenApiRequestValidationSchema, {
          specYaml: new TextEncoder().encode(c.specYaml),
        }),
      };
      break;
    }
  }

  return policy;
}

function toProtobufMatch(m: MatchFormData) {
  const expr = create(MatchExprSchema);
  switch (m.type) {
    case "path":
      expr.expr = {
        case: "path",
        value: create(PathMatchSchema, {
          path: createStringMatch(m.pathMode ?? "prefix", m.pathValue ?? "", m.pathIgnoreCase),
        }),
      };
      break;
    case "method":
      expr.expr = {
        case: "method",
        value: create(MethodMatchSchema, { methods: m.methods ?? [] }),
      };
      break;
    case "header":
      expr.expr = {
        case: "header",
        value: create(HeaderMatchSchema, {
          name: m.headerName ?? "",
          match: m.headerPresent
            ? { case: "present", value: true }
            : {
                case: "value",
                value: createStringMatch(
                  m.headerMode ?? "exact",
                  m.headerValue ?? "",
                  m.headerIgnoreCase,
                ),
              },
        }),
      };
      break;
    case "queryParam":
      expr.expr = {
        case: "queryParam",
        value: create(QueryParamMatchSchema, {
          name: m.queryParamName ?? "",
          match: m.queryParamPresent
            ? { case: "present", value: true }
            : {
                case: "value",
                value: createStringMatch(
                  m.queryParamMode ?? "exact",
                  m.queryParamValue ?? "",
                  m.queryParamIgnoreCase,
                ),
              },
        }),
      };
      break;
  }
  return expr;
}

function createStringMatch(mode: StringMatchMode, value: string, ignoreCase?: boolean) {
  return create(StringMatchSchema, {
    ignoreCase: ignoreCase ?? false,
    match: { case: mode, value },
  });
}

// ── Protobuf → Form data ──────────────────────────────────────────────

export function fromProtobufPolicy(p: Policy): PolicyFormData {
  const type = (p.config.case ?? "keyauth") as PolicyType;
  return {
    id: p.id,
    name: p.name,
    enabled: p.enabled,
    type,
    match: p.match.map(fromProtobufMatch),
    config: fromProtobufConfig(p),
  };
}

function fromProtobufMatch(m: { expr: { case?: string; value?: unknown } }): MatchFormData {
  const id = crypto.randomUUID();
  const e = m.expr;
  switch (e.case) {
    case "path": {
      const v = e.value as {
        path?: { match?: { case?: string; value?: string }; ignoreCase?: boolean };
      };
      return {
        id,
        type: "path",
        pathMode: (v.path?.match?.case as StringMatchMode) ?? "prefix",
        pathValue: (v.path?.match?.value as string) ?? "",
        pathIgnoreCase: v.path?.ignoreCase ?? false,
      };
    }
    case "method": {
      const v = e.value as { methods?: string[] };
      return { id, type: "method", methods: v.methods ?? [] };
    }
    case "header": {
      const v = e.value as {
        name?: string;
        match?: {
          case?: string;
          value?: boolean | { match?: { case?: string; value?: string }; ignoreCase?: boolean };
        };
      };
      if (v.match?.case === "present") {
        return { id, type: "header", headerName: v.name ?? "", headerPresent: true };
      }
      const sv = v.match?.value as
        | { match?: { case?: string; value?: string }; ignoreCase?: boolean }
        | undefined;
      return {
        id,
        type: "header",
        headerName: v.name ?? "",
        headerPresent: false,
        headerMode: (sv?.match?.case as StringMatchMode) ?? "exact",
        headerValue: (sv?.match?.value as string) ?? "",
        headerIgnoreCase: sv?.ignoreCase ?? false,
      };
    }
    case "queryParam": {
      const v = e.value as {
        name?: string;
        match?: {
          case?: string;
          value?: boolean | { match?: { case?: string; value?: string }; ignoreCase?: boolean };
        };
      };
      if (v.match?.case === "present") {
        return { id, type: "queryParam", queryParamName: v.name ?? "", queryParamPresent: true };
      }
      const sv = v.match?.value as
        | { match?: { case?: string; value?: string }; ignoreCase?: boolean }
        | undefined;
      return {
        id,
        type: "queryParam",
        queryParamName: v.name ?? "",
        queryParamPresent: false,
        queryParamMode: (sv?.match?.case as StringMatchMode) ?? "exact",
        queryParamValue: (sv?.match?.value as string) ?? "",
        queryParamIgnoreCase: sv?.ignoreCase ?? false,
      };
    }
    default:
      return { id, type: "path" };
  }
}

function fromProtobufConfig(p: Policy): PolicyFormData["config"] {
  switch (p.config.case) {
    case "keyauth": {
      const v = p.config.value;
      return {
        keySpaceIds: v.keySpaceIds,
        locations: v.locations.map((loc) => {
          switch (loc.location.case) {
            case "bearer":
              return { type: "bearer" as const };
            case "header":
              return {
                type: "header" as const,
                name: loc.location.value.name,
                stripPrefix: loc.location.value.stripPrefix,
              };
            case "queryParam":
              return { type: "queryParam" as const, name: loc.location.value.name };
            default:
              return { type: "bearer" as const };
          }
        }),
        permissionQuery: v.permissionQuery ?? "",
      } satisfies KeyAuthConfig;
    }
    case "jwtauth": {
      const v = p.config.value;
      return {
        jwksSource: (v.jwksSource.case ?? "oidcIssuer") as JWTAuthConfig["jwksSource"],
        jwksValue:
          v.jwksSource.case === "publicKeyPem"
            ? new TextDecoder().decode(v.jwksSource.value)
            : ((v.jwksSource.value as string) ?? ""),
        issuer: v.issuer,
        audiences: v.audiences,
        algorithms: v.algorithms,
        subjectClaim: v.subjectClaim || "sub",
        forwardClaims: v.forwardClaims,
        allowAnonymous: v.allowAnonymous,
      } satisfies JWTAuthConfig;
    }
    case "basicauth": {
      const v = p.config.value;
      return {
        credentials: v.credentials.map((c) => ({
          username: c.username,
          passwordHash: c.passwordHash,
        })),
      } satisfies BasicAuthConfig;
    }
    case "ratelimit": {
      const v = p.config.value;
      let keySource: RateLimitConfig["keySource"] = "remoteIp";
      let keyValue = "";
      if (v.key) {
        switch (v.key.source.case) {
          case "remoteIp":
            keySource = "remoteIp";
            break;
          case "header":
            keySource = "header";
            keyValue = v.key.source.value.name;
            break;
          case "authenticatedSubject":
            keySource = "authenticatedSubject";
            break;
          case "path":
            keySource = "path";
            break;
          case "principalClaim":
            keySource = "principalClaim";
            keyValue = v.key.source.value.claimName;
            break;
        }
      }
      return {
        limit: Number(v.limit),
        windowMs: Number(v.windowMs),
        keySource,
        keyValue,
      } satisfies RateLimitConfig;
    }
    case "ipRules": {
      const v = p.config.value;
      return { allow: [...v.allow], deny: [...v.deny] } satisfies IPRulesConfig;
    }
    case "openapi": {
      const v = p.config.value;
      return {
        specYaml: new TextDecoder().decode(v.specYaml),
      } satisfies OpenAPIConfig;
    }
    default:
      return { keySpaceIds: [], locations: [], permissionQuery: "" } satisfies KeyAuthConfig;
  }
}
