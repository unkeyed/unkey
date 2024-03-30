import { t } from "../trpc";
import { createApi } from "./api/create";
import { deleteApi } from "./api/delete";
import { updateApiIpWhitelist } from "./api/updateIpWhitelist";
import { updateApiName } from "./api/updateName";
import { createGateway } from "./gateway/create";
import { createKey } from "./key/create";
import { createRootKey } from "./key/createRootKey";
import { deleteKeys } from "./key/delete";
import { deleteRootKeys } from "./key/deleteRootKey";
import { updateKeyEnabled } from "./key/updateEnabled";
import { updateKeyExpiration } from "./key/updateExpiration";
import { updateKeyMetadata } from "./key/updateMetadata";
import { updateKeyName } from "./key/updateName";
import { updateKeyOwnerId } from "./key/updateOwnerId";
import { updateKeyRatelimit } from "./key/updateRatelimit";
import { updateKeyRemaining } from "./key/updateRemaining";
import { createPlainIssue } from "./plain";
import { createNamespace } from "./ratelimit/createNamespace";
import { createOverride } from "./ratelimit/createOverride";
import { deleteNamespace } from "./ratelimit/deleteNamespace";
import { deleteOverride } from "./ratelimit/deleteOverride";
import { updateNamespaceName } from "./ratelimit/updateNamespaceName";
import { updateOverride } from "./ratelimit/updateOverride";
import { addPermissionToRootKey } from "./rbac/addPermissionToRootKey";
import { connectPermissionToRole } from "./rbac/connectPermissionToRole";
import { connectRoleToKey } from "./rbac/connectRoleToKey";
import { createPermission } from "./rbac/createPermission";
import { createRole } from "./rbac/createRole";
import { deletePermission } from "./rbac/deletePermission";
import { deleteRole } from "./rbac/deleteRole";
import { disconnectPermissionFromRole } from "./rbac/disconnectPermissionFromRole";
import { disconnectRoleFromKey } from "./rbac/disconnectRoleFromKey";
import { removePermissionFromRootKey } from "./rbac/removePermissionFromRootKey";
import { updatePermission } from "./rbac/updatePermission";
import { updateRole } from "./rbac/updateRole";
import { createSecret } from "./secrets/create";
import { decryptSecret } from "./secrets/decrypt";
import { updateSecret } from "./secrets/update";
import { vercelRouter } from "./vercel";
import { changeWorkspaceName } from "./workspace/changeName";
import { changeWorkspacePlan } from "./workspace/changePlan";
import { createWorkspace } from "./workspace/create";
import { optWorkspaceIntoBeta } from "./workspace/optIntoBeta";

export const router = t.router({
  key: t.router({
    create: createKey,
    delete: deleteKeys,
    update: t.router({
      enabled: updateKeyEnabled,
      expiration: updateKeyExpiration,
      metadata: updateKeyMetadata,
      name: updateKeyName,
      ownerId: updateKeyOwnerId,
      ratelimit: updateKeyRatelimit,
      remaining: updateKeyRemaining,
    }),
  }),
  gateway: t.router({
    create: createGateway,
  }),
  secrets: t.router({
    create: createSecret,
    decrypt: decryptSecret,
    update: updateSecret,
  }),

  rootKey: t.router({
    create: createRootKey,
    delete: deleteRootKeys,
  }),
  api: t.router({
    create: createApi,
    delete: deleteApi,
    updateName: updateApiName,
    updateIpWhitelist: updateApiIpWhitelist,
  }),
  workspace: t.router({
    create: createWorkspace,
    updateName: changeWorkspaceName,
    updatePlan: changeWorkspacePlan,
    optIntoBeta: optWorkspaceIntoBeta,
  }),
  vercel: vercelRouter,
  plain: t.router({
    createIssue: createPlainIssue,
  }),
  rbac: t.router({
    addPermissionToRootKey: addPermissionToRootKey,
    connectPermissionToRole: connectPermissionToRole,
    connectRoleToKey: connectRoleToKey,
    createPermission: createPermission,
    createRole: createRole,
    deletePermission: deletePermission,
    deleteRole: deleteRole,
    disconnectPermissionFromRole: disconnectPermissionFromRole,
    disconnectRoleFromKey: disconnectRoleFromKey,
    removePermissionFromRootKey: removePermissionFromRootKey,
    updatePermission: updatePermission,
    updateRole: updateRole,
  }),
  ratelimit: t.router({
    namespace: t.router({
      create: createNamespace,
      update: t.router({
        name: updateNamespaceName,
      }),
      delete: deleteNamespace,
    }),
    override: t.router({
      create: createOverride,
      update: updateOverride,
      delete: deleteOverride,
    }),
  }),
});

// export type definition of API
export type Router = typeof router;
