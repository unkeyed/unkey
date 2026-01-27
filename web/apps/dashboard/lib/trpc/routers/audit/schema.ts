import { AUDIT_LOG_BUCKET } from "@/lib/audit";
import { z } from "zod";

export const auditLog = z.object({
  user: z
    .object({
      username: z.string().nullable(),
      firstName: z.string().nullable(),
      lastName: z.string().nullable(),
      imageUrl: z.string().nullable(),
    })
    .optional(),
  auditLog: z.object({
    id: z.string(),
    time: z.number(),
    actor: z.object({
      id: z.string(),
      name: z.string().nullable(),
      type: z.string(),
    }),
    location: z.string().nullable(),
    description: z.string(),
    userAgent: z.string().nullable(),
    event: z.string(),
    workspaceId: z.string(),
    targets: z.array(
      z.object({
        id: z.string(),
        type: z.string(),
        name: z.string().nullable(),
        meta: z.unknown(),
      }),
    ),
  }),
});

export type AuditLog = z.infer<typeof auditLog>;

export type AuditLogWithTargets = {
  workspaceId: string;
  id: string;
  bucket: string;
  createdAt: number;
  updatedAt: number | null;
  time: number;
  event: string;
  display: string;
  remoteIp: string | null;
  userAgent: string | null;
  actorType: string;
  actorId: string;
  actorName: string | null;
  actorMeta: unknown;
} & {
  targets: Array<{
    workspaceId: string;
    bucket: string;
    auditLogId: string;
    displayName: string;
    type: string;
    id: string;
    name: string | null;
    meta: unknown;
    createdAt: number;
    updatedAt: number | null;
  }>;
};

export const auditQueryLogsParamsSchema = z.object({
  workspaceId: z.string(),
  bucket: z.string().prefault(AUDIT_LOG_BUCKET),
  limit: z.int(),
  startTime: z.int().optional(),
  endTime: z.int().optional(),
  events: z
    .array(
      z.object({
        operator: z.literal("is"),
        value: z.string(),
      }),
    )
    .nullable(),
  users: z
    .array(
      z.object({
        operator: z.literal("is"),
        value: z.string(),
      }),
    )
    .nullable(),
  rootKeys: z
    .array(
      z.object({
        operator: z.literal("is"),
        value: z.string(),
      }),
    )
    .nullable(),
  cursor: z.int().nullable().optional(),
});

export type AuditQueryLogsParams = z.infer<typeof auditQueryLogsParamsSchema>;
