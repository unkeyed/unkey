import { type Result, result } from "@unkey/result";
import { NestedQuery, RoleQuery } from "./queries";
export class RBAC {
  public evaluateRoles(
    rq: RoleQuery,
    roles: string[],
  ): Result<{ valid: true } | { valid: false; message: string }> {
    if (rq.version !== 1) {
      return result.fail({ valid: false, message: "invalid version, only version 1 is supported" });
    }
    return this.evaluateNestedQueryV1(rq.query, roles);
  }

  private evaluateNestedQueryV1(
    query: NestedQuery,
    roles: string[],
  ): Result<{ valid: true } | { valid: false; message: string }> {
    if (typeof query === "string") {
      // Check if the role is in the list of roles
      if (roles.includes(query)) {
        return result.success({ valid: true });
      } else {
        return result.success({ valid: false, message: `Role ${query} not allowed` });
      }
    }

    if (query.and) {
      const results = query.and.map((q) => this.evaluateNestedQueryV1(q, roles));
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
        const r = this.evaluateNestedQueryV1(q, roles);
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
