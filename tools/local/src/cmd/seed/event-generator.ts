import crypto from "node:crypto";
import {
  generateMetadata,
  generateRandomApiRequest,
  generateRandomString,
  generateTimestamp,
  generateUuid,
  getNormallyDistributedIndex,
  identifierPools,
  initializeIdentifierPools,
  selectIdentifier,
} from "./utils";

// Types for event generation
export type KeyInfo = {
  id: string;
  name: string;
  prefix: string;
  enabled: boolean;
  hasRatelimit: boolean;
  hasUsageLimit: boolean;
};

export type VerificationEvent = {
  request_id: string;
  time: number;
  workspace_id: string;
  key_space_id: string;
  key_id: string;
  region: string;
  tags: string[];
  outcome: string;
  identity_id: string;
};

export type RatelimitEvent = {
  request_id: string;
  time: number;
  workspace_id: string;
  namespace_id: string;
  identifier: string;
  passed: boolean;
  remaining: number;
  limit: number;
  reset: number;
};

export type ApiRequestEvent = {
  request_id: string;
  time: number;
  path: string;
  request_body?: string;
  response_status: number;
  response_body?: string;
  error?: string;
  [key: string]: unknown;
};

/**
 * Selects a key with normal distribution to create realistic "hot key" patterns
 */
export function selectKeyWithNormalDistribution(keys: KeyInfo[]): KeyInfo {
  const keyCount = keys.length;

  // Center the mean at 20% of the array to make some keys "hotter" than others
  const mean = Math.floor(keyCount * 0.2);

  // Standard deviation - adjust to control how concentrated verifications are
  const stdDev = keyCount / 5; // Standard deviation of 20% of the array size

  // Use the normal distribution function
  const index = getNormallyDistributedIndex(mean, stdDev, 0, keyCount);

  return keys[index];
}

/**
 * Generates a hash for a key
 */
export function generateKeyHash(keyContent: string): string {
  return crypto.createHash("sha256").update(keyContent).digest("hex");
}

/**
 * Generates a random key name with realistic patterns
 */
export function generateKeyName(): string {
  const environments = ["Development", "Staging", "Production", "Testing"];
  const purposes = [
    "API Access",
    "Admin",
    "Readonly",
    "Server",
    "Client",
    "Backend",
    "Frontend",
    "Mobile",
  ];
  const environment = environments[Math.floor(Math.random() * environments.length)];
  const purpose = purposes[Math.floor(Math.random() * purposes.length)];

  if (Math.random() > 0.5) {
    return `${environment} ${purpose} Key`;
  }
  return `${purpose} Key for ${environment}`;
}

/**
 * Generates a verification event with realistic distribution of outcomes
 */
export function generateVerificationEvent(
  workspaceId: string,
  keyspaceId: string,
  keyId: string,
  requestId?: string,
): VerificationEvent {
  // Generate a timestamp using the same distribution logic as API requests
  const time = generateTimestamp();

  // Outcomes with realistic distribution
  const outcomeDistribution = Math.random();
  let outcome: string;

  if (outcomeDistribution < 0.85) {
    outcome = "VALID"; // 85% valid
  } else if (outcomeDistribution < 0.9) {
    outcome = "RATE_LIMITED"; // 5% rate limited
  } else if (outcomeDistribution < 0.93) {
    outcome = "EXPIRED"; // 3% expired
  } else if (outcomeDistribution < 0.95) {
    outcome = "DISABLED"; // 2% disabled
  } else if (outcomeDistribution < 0.97) {
    outcome = "FORBIDDEN"; // 2% forbidden
  } else if (outcomeDistribution < 0.99) {
    outcome = "USAGE_EXCEEDED"; // 2% usage exceeded
  } else {
    outcome = "INSUFFICIENT_PERMISSIONS"; // 1% insufficient permissions
  }

  // Generate tags (common categories for segmenting key verifications)
  const tagOptions = [
    "api",
    "web",
    "mobile",
    "server",
    "client",
    "frontend",
    "backend",
    "test",
    "prod",
  ];

  const tagCount = Math.floor(Math.random() * 3); // 0-2 tags
  const tags: string[] = [];

  for (let i = 0; i < tagCount; i++) {
    const tag = tagOptions[Math.floor(Math.random() * tagOptions.length)];
    if (!tags.includes(tag)) {
      tags.push(tag);
    }
  }

  // Sort tags to match expected format
  tags.sort();

  // Generate request ID if not provided
  const generatedRequestId = requestId || generateUuid();

  // Generate a region
  const regions = ["us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1", "sa-east-1"];

  const region = regions[Math.floor(Math.random() * regions.length)];

  // Optionally include identity_id (30% of the time)
  const identityId = Math.random() < 0.3 ? `ident_${generateRandomString(24)}` : "";

  return {
    request_id: generatedRequestId,
    time,
    workspace_id: workspaceId,
    key_space_id: keyspaceId,
    key_id: keyId,
    region,
    tags,
    outcome,
    identity_id: identityId,
  };
}

/**
 * Biases verification event outcome based on key properties
 */
export function biasVerificationOutcome(
  key: KeyInfo,
  workspaceId: string,
  keyAuthId: string,
  requestId: string,
): VerificationEvent {
  let verificationEvent: VerificationEvent;

  // If key is disabled, higher chance of DISABLED outcome
  if (!key.enabled && Math.random() < 0.6) {
    verificationEvent = generateVerificationEvent(workspaceId, keyAuthId, key.id, requestId);
    verificationEvent.outcome = "DISABLED";
  }
  // If key has rate limit, higher chance of RATE_LIMITED outcome
  else if (key.hasRatelimit && Math.random() < 0.3) {
    verificationEvent = generateVerificationEvent(workspaceId, keyAuthId, key.id, requestId);
    verificationEvent.outcome = "RATE_LIMITED";
  }
  // If key has usage limit, higher chance of USAGE_EXCEEDED outcome
  else if (key.hasUsageLimit && Math.random() < 0.2) {
    verificationEvent = generateVerificationEvent(workspaceId, keyAuthId, key.id, requestId);
    verificationEvent.outcome = "USAGE_EXCEEDED";
  }
  // Otherwise generate a regular event
  else {
    verificationEvent = generateVerificationEvent(workspaceId, keyAuthId, key.id, requestId);
  }

  return verificationEvent;
}

/**
 * Generates a ratelimit event with realistic distributions
 */
export function generateRatelimitEvent(
  workspaceId: string,
  namespaceId: string,
  requestId?: string,
): RatelimitEvent {
  // Generate a random timestamp with the same distribution as API requests
  const time = generateTimestamp();

  // Ensure pools are initialized
  if (Object.keys(identifierPools).length === 0) {
    initializeIdentifierPools();
  }

  const identifier = selectIdentifier();

  // Determine if this request passed the rate limit
  const passed = Math.random() < 0.9; // 90% of requests pass rate limiting

  // Random values for rate limit tracking
  const remaining = passed ? Math.floor(Math.random() * 1000) : 0;
  const limit = 500 + Math.floor(Math.random() * 9500); // Between 500 and 10000
  const reset = Date.now() + 60 * 1000; // Reset in 1 minute

  // Generate a request ID if not provided
  const generated_request_id = requestId || generateUuid();

  return {
    request_id: generated_request_id,
    time,
    workspace_id: workspaceId,
    namespace_id: namespaceId,
    identifier,
    passed,
    remaining,
    limit,
    reset,
  };
}

/**
 * Generates API request event that corresponds to a verification event
 */
export function generateMatchingApiRequestForVerification(
  workspaceId: string,
  verificationEvent: VerificationEvent,
  keyPrefix: string,
): ApiRequestEvent {
  const apiRequest = generateRandomApiRequest(workspaceId);

  // Match the verification event
  apiRequest.request_id = verificationEvent.request_id;
  apiRequest.time = verificationEvent.time;

  // Ensure we're using the key verification endpoint
  apiRequest.path = "/v1/keys.verify";

  // Set key information in request body
  const requestBody = {
    key: `${keyPrefix}_${generateRandomString(32)}`,
    apiId: `api_${generateRandomString(24)}`,
  };

  apiRequest.request_body = JSON.stringify(requestBody);

  // Set response status based on verification outcome
  if (verificationEvent.outcome !== "VALID") {
    switch (verificationEvent.outcome) {
      case "RATE_LIMITED":
        apiRequest.response_status = 429; // Too Many Requests
        apiRequest.error = "Rate limit exceeded";
        break;
      case "EXPIRED":
      case "DISABLED":
      case "FORBIDDEN":
      case "USAGE_EXCEEDED":
      case "INSUFFICIENT_PERMISSIONS":
        apiRequest.response_status = 401; // Unauthorized
        apiRequest.error = verificationEvent.outcome.toLowerCase().replace("_", " ");
        break;
    }

    apiRequest.response_body = JSON.stringify({
      valid: false,
      code: String(apiRequest.response_status),
      message: apiRequest.error,
    });
  } else {
    // Valid response
    apiRequest.response_status = 200;
    apiRequest.response_body = JSON.stringify({
      valid: true,
      keyId: verificationEvent.key_id,
      ownerId: `user_${generateRandomString(16)}`,
      meta: Math.random() < 0.7 ? generateMetadata() : undefined,
    });
  }

  return apiRequest;
}

/**
 * Generates API request event that corresponds to a ratelimit event
 */
export function generateMatchingApiRequestForRatelimit(
  workspaceId: string,
  ratelimitEvent: RatelimitEvent,
): ApiRequestEvent {
  const apiRequest = generateRandomApiRequest(workspaceId);

  // Match the ratelimit event
  apiRequest.request_id = ratelimitEvent.request_id;
  apiRequest.time = ratelimitEvent.time;

  // Set response status based on if it passed rate limiting
  if (!ratelimitEvent.passed) {
    apiRequest.response_status = 429; // Too Many Requests
    apiRequest.error = "Rate limit exceeded";
    apiRequest.response_body = JSON.stringify({
      error: {
        code: "rate_limit_exceeded",
        message: "You have exceeded the rate limit for this endpoint",
        details: {
          limit: ratelimitEvent.limit,
          reset: ratelimitEvent.reset,
        },
      },
    });
  }

  return apiRequest;
}
