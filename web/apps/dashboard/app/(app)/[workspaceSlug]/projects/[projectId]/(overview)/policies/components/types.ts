export type PolicyType = "keyauth" | "jwtauth" | "basicauth" | "ratelimit" | "ipRules" | "openapi";

export type MatchType = "path" | "method" | "header" | "queryParam";

export type StringMatchMode = "exact" | "prefix" | "regex";

export type MatchFormData = {
  id: string;
  type: MatchType;
  // path
  pathMode?: StringMatchMode;
  pathValue?: string;
  pathIgnoreCase?: boolean;
  // method
  methods?: string[];
  // header
  headerName?: string;
  headerPresent?: boolean;
  headerMode?: StringMatchMode;
  headerValue?: string;
  headerIgnoreCase?: boolean;
  // queryParam
  queryParamName?: string;
  queryParamPresent?: boolean;
  queryParamMode?: StringMatchMode;
  queryParamValue?: string;
  queryParamIgnoreCase?: boolean;
};

export type KeyAuthConfig = {
  keySpaceIds: string[];
  locations: Array<
    | { type: "bearer" }
    | { type: "header"; name: string; stripPrefix?: string }
    | { type: "queryParam"; name: string }
  >;
  permissionQuery: string;
};

export type JWTAuthConfig = {
  jwksSource: "jwksUri" | "oidcIssuer" | "publicKeyPem";
  jwksValue: string;
  issuer: string;
  audiences: string[];
  algorithms: string[];
  subjectClaim: string;
  forwardClaims: string[];
  allowAnonymous: boolean;
};

export type BasicAuthConfig = {
  credentials: Array<{ username: string; passwordHash: string }>;
};

export type RateLimitConfig = {
  limit: number;
  windowMs: number;
  keySource: "remoteIp" | "header" | "authenticatedSubject" | "path" | "principalClaim";
  keyValue: string; // header name or claim name, depending on keySource
};

export type IPRulesConfig = {
  allow: string[];
  deny: string[];
};

export type OpenAPIConfig = {
  specYaml: string;
};

export type PolicyFormData = {
  id: string;
  name: string;
  enabled: boolean;
  type: PolicyType;
  match: MatchFormData[];
  config:
    | KeyAuthConfig
    | JWTAuthConfig
    | BasicAuthConfig
    | RateLimitConfig
    | IPRulesConfig
    | OpenAPIConfig;
};
