export type ExternalIdBackfillRow = {
  workspace_id: string;
  key_space_id: string;
  identity_id: string;
};

const TABLE_NAME = /^[A-Za-z0-9_]+\.[A-Za-z0-9_]+$/;

/**
 * Builds the ClickHouse mutation that backfills `external_id` for rows whose
 * identity has since been given one. Every caller-supplied value is passed as a
 * query parameter so it is substituted by ClickHouse rather than interpolated
 * into the statement text.
 */
export function buildExternalIdBackfillMutation(
  table: string,
  row: ExternalIdBackfillRow,
  externalId: string,
): { query: string; query_params: Record<string, string> } {
  if (!TABLE_NAME.test(table)) {
    throw new Error(`Refusing to interpolate unsupported table name "${table}"`);
  }

  return {
    query: `
    UPDATE ${table}
    SET external_id = {externalId: String}
    WHERE
    workspace_id = {workspaceId: String}
    AND key_space_id = {keySpaceId: String}
    AND identity_id = {identityId: String}
    AND ( external_id = '' OR external_id = 'undefined' )
    `,
    query_params: {
      externalId,
      workspaceId: row.workspace_id,
      keySpaceId: row.key_space_id,
      identityId: row.identity_id,
    },
  };
}
