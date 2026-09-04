import { describe, expect, it, vi } from "vitest";
import { ROOT_KEY_COLUMN_IDS, createRootKeyColumns } from "./create-root-key-columns";

describe("createRootKeyColumns", () => {
  it("includes a sortable Last Used column", () => {
    const columns = createRootKeyColumns({ onEditKey: vi.fn() });

    expect(ROOT_KEY_COLUMN_IDS.LAST_USED).toEqual({
      id: "last_used",
      accessorKey: "last_used",
      header: "Last Used",
      emptyText: "—",
    });
    expect(columns).toContainEqual(
      expect.objectContaining({
        id: "last_used",
        accessorKey: "last_used",
        sortDescFirst: true,
      }),
    );
  });

  it("does not claim a root key was never used when no usage was recorded", () => {
    expect(ROOT_KEY_COLUMN_IDS.LAST_USED.emptyText).toBe("—");
  });
});
