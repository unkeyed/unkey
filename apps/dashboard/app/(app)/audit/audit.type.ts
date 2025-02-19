export type AuditData = {
  user:
    | {
        username?: string | null;
        firstName?: string | null;
        lastName?: string | null;
        imageUrl?: string | null;
      }
    | undefined;
  auditLog: {
    id: string;
    time: number;
    actor: {
      id: string;
      type: string;
      name: string | null;
    };
    event: string;
    location: string | null;
    userAgent: string | null;
    workspaceId: string | null;
    targets: Array<{
      id: string;
      type: string;
      name: string | null;
      meta: unknown;
    }>;
    description: string;
  };
};
