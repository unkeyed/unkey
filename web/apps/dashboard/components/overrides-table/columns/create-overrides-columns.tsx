import type { RatelimitOverride } from "@/lib/collections";
import type { DataTableColumnDef } from "@unkey/ui";
import { OverridesTableAction } from "../components/actions/overrides-table-action";
import { LastUsedCell } from "../components/cells/last-used-cell";
import { OverrideIdCell } from "../components/cells/override-id-cell";
import { OverrideIdentifierCell } from "../components/cells/override-identifier-cell";
import { OverrideLimitsCell } from "../components/cells/override-limits-cell";

export const OVERRIDE_COLUMN_IDS = {
  ID: "id",
  IDENTIFIER: "identifier",
  LIMITS: "limits",
  LAST_USED: "lastUsed",
  ACTIONS: "actions",
} as const;

type CreateOverridesColumnsOptions = {
  namespaceId: string;
};

export const createOverridesColumns = ({
  namespaceId,
}: CreateOverridesColumnsOptions): DataTableColumnDef<RatelimitOverride>[] => [
  {
    id: OVERRIDE_COLUMN_IDS.ID,
    header: "ID",
    enableSorting: false,
    meta: {
      width: "20%",
      headerClassName: "pl-2",
    },
    cell: ({ row }) => {
      return <OverrideIdCell id={row.original.id} />;
    },
  },
  {
    id: OVERRIDE_COLUMN_IDS.IDENTIFIER,
    header: "Identifier",
    enableSorting: false,
    meta: {
      width: "auto",
      headerClassName: "pl-2",
    },
    cell: ({ row }) => {
      return <OverrideIdentifierCell identifier={row.original.identifier} />;
    },
  },
  {
    id: OVERRIDE_COLUMN_IDS.LIMITS,
    header: "Limits",
    enableSorting: false,
    meta: {
      width: "10%",
    },
    cell: ({ row }) => {
      return <OverrideLimitsCell limit={row.original.limit} duration={row.original.duration} />;
    },
  },
  {
    id: OVERRIDE_COLUMN_IDS.LAST_USED,
    header: "Last used",
    enableSorting: false,
    meta: {
      width: { min: 150, max: 200 },
    },
    cell: ({ row }) => {
      return <LastUsedCell namespaceId={namespaceId} identifier={row.original.identifier} />;
    },
  },
  {
    id: OVERRIDE_COLUMN_IDS.ACTIONS,
    header: "",
    enableSorting: false,
    meta: {
      width: "10%",
    },
    cell: ({ row }) => {
      const override = row.original;
      return (
        <OverridesTableAction
          overrideDetails={{
            duration: override.duration,
            limit: override.limit,
            overrideId: override.id,
          }}
          identifier={override.identifier}
          namespaceId={namespaceId}
        />
      );
    },
  },
];
