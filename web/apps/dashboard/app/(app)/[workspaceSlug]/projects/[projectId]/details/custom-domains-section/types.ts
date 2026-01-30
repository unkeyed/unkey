export type VerificationStatus = "pending" | "verifying" | "verified" | "failed";

export type CustomDomain = {
  id: string;
  domain: string;
  workspaceId: string;
  projectId: string;
  environmentId: string;
  verificationStatus: VerificationStatus;
  verificationToken: string;
  ownershipVerified: boolean;
  cnameVerified: boolean;
  targetCname: string;
  checkAttempts: number;
  lastCheckedAt: number | null;
  verificationError: string | null;
  createdAt: number;
  updatedAt: number | null;
};
