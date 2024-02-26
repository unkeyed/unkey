import { type Result, result } from "@unkey/result";
import { generateErrorMessage } from "zod-error";
import { type PermissionQuery, permissionQuerySchema } from "./queries";
export class RBAC {
  public evaluatePermissions(
    q: PermissionQuery,
    roles: string[],
  ): Result<{ valid: true } | { valid: false; message: string }> {
    return this.evaluateQueryV1(q, roles);
  }
  public validateQuery(q: PermissionQuery): Result<{ query: PermissionQuery }> {
    const validQuery = permissionQuerySchema.safeParse(q);
    if (!validQuery.success) {
      const message = generateErrorMessage(validQuery.error.issues, {
        maxErrors: 1,
        delimiter: {
          component: ": ",
        },
        path: {
          enabled: true,
          type: "objectNotation",
          label: "",
        },
        code: {
          enabled: true,
          label: "",
        },
        message: {
          enabled: true,
          label: "",
        },
      });
      return result.fail({ valid: false, message });
    }

    return result.success({ query: validQuery.data });
  }

  private evaluateQueryV1(
    query: PermissionQuery,
    roles: string[],
  ): Result<{ valid: true } | { valid: false; message: string }> {
    if (typeof query === "string") {
      // Check if the role is in the list of roles
      if (roles.includes(query)) {
        return result.success({ valid: true });
      }
      return result.success({ valid: false, message: `Role ${query} not allowed` });
    }

    if (query.and) {
      const results = query.and.map((q) => this.evaluateQueryV1(q, roles));
      for (const r of results) {
        if (r.error) {
          return r;
        }
        if (!r.value.valid) {
          return r;
        }
      }
      return result.success({ valid: true });
    }

    if (query.or) {
      for (const q of query.or) {
        const r = this.evaluateQueryV1(q, roles);
        if (r.error) {
          return r;
        }
        if (r.value.valid) {
          return r;
        }
      }
      return result.success({ valid: false, message: "No role matched" });
    }

    return result.fail({ message: "reached end of evaluate and no match" });
  }
}
