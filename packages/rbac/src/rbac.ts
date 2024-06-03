import { Err, Ok, type Result, SchemaError } from "@unkey/error";
import { type PermissionQuery, permissionQuerySchema } from "./queries";

export class RBAC {
  public evaluatePermissions(
    q: PermissionQuery,
    roles: string[],
  ): Result<{ valid: true; message?: never } | { valid: false; message: string }, SchemaError> {
    return this.evaluateQueryV1(q, roles);
  }
  public validateQuery(q: PermissionQuery): Result<{ query: PermissionQuery }> {
    const validQuery = permissionQuerySchema.safeParse(q);
    if (!validQuery.success) {
      return Err(SchemaError.fromZod(validQuery.error, q));
    }

    return Ok({ query: validQuery.data });
  }

  private evaluateQueryV1(
    query: PermissionQuery,
    roles: string[],
  ): Result<{ valid: true; message?: never } | { valid: false; message: string }, SchemaError> {
    if (typeof query === "string") {
      // Check if the role is in the list of roles
      if (roles.includes(query)) {
        return Ok({ valid: true });
      }
      return Ok({ valid: false, message: `Role ${query} not allowed` });
    }

    if (query.and) {
      const results = query.and
        .filter(Boolean)
        .map((q) => this.evaluateQueryV1(q as Required<PermissionQuery>, roles));
      for (const r of results) {
        if (r.err) {
          return r;
        }
        if (!r.val.valid) {
          return r;
        }
      }
      return Ok({ valid: true });
    }

    if (query.or) {
      for (const q of query.or) {
        const r = this.evaluateQueryV1(q as Required<PermissionQuery>, roles);
        if (r.err) {
          return r;
        }
        if (r.val.valid) {
          return r;
        }
      }
      return Ok({ valid: false, message: "No role matched" });
    }

    return Err(new SchemaError({ message: "reached end of evaluate and no match" }));
  }
}
