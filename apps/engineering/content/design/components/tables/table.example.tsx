"use client";
import { RenderComponentWithSnippet } from "@/app/components/render";
import { type ColumnDef, Table } from "@unkey/ui";

// Sample data types
interface User {
  id: string;
  name: string;
  email: string;
  role: string;
  status: "active" | "inactive";
  lastSeen: Date;
}

interface ApiKey {
  id: string;
  name: string;
  prefix: string;
  created: Date;
  requests: number;
  rateLimit: string;
}

// Sample data
const sampleUsers: User[] = [
  {
    id: "1",
    name: "Alice Johnson",
    email: "alice@example.com",
    role: "Admin",
    status: "active",
    lastSeen: new Date("2024-11-24"),
  },
  {
    id: "2",
    name: "Bob Smith",
    email: "bob@example.com",
    role: "Developer",
    status: "active",
    lastSeen: new Date("2024-11-23"),
  },
  {
    id: "3",
    name: "Carol White",
    email: "carol@example.com",
    role: "Viewer",
    status: "inactive",
    lastSeen: new Date("2024-11-20"),
  },
  {
    id: "4",
    name: "David Brown",
    email: "david@example.com",
    role: "Developer",
    status: "active",
    lastSeen: new Date("2024-11-24"),
  },
  {
    id: "5",
    name: "Eve Davis",
    email: "eve@example.com",
    role: "Admin",
    status: "active",
    lastSeen: new Date("2024-11-22"),
  },
];

const sampleApiKeys: ApiKey[] = Array.from({ length: 50 }, (_, i) => ({
  id: `key_${i + 1}`,
  name: `API Key ${i + 1}`,
  prefix: `uk_${Math.random().toString(36).substring(7)}`,
  created: new Date(Date.now() - Math.random() * 90 * 24 * 60 * 60 * 1000),
  requests: Math.floor(Math.random() * 10000),
  rateLimit: `${Math.floor(Math.random() * 100 + 10)}/min`,
}));

export function BasicTable() {
  const columns: ColumnDef<User>[] = [
    {
      accessorKey: "name",
      header: "Name",
      size: 200,
    },
    {
      accessorKey: "email",
      header: "Email",
      size: 250,
    },
    {
      accessorKey: "role",
      header: "Role",
      size: 120,
    },
    {
      accessorKey: "status",
      header: "Status",
      size: 100,
      cell: ({ row }) => (
        <span
          className={`px-2 py-1 rounded text-xs ${
            row.getValue("status") === "active"
              ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300"
              : "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300"
          }`}
        >
          {row.getValue("status") as string}
        </span>
      ),
    },
  ];

  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`import { Table, type ColumnDef } from "@unkey/ui";

const columns: ColumnDef<User>[] = [
  { accessorKey: "name", header: "Name" },
  { accessorKey: "email", header: "Email" },
  { accessorKey: "role", header: "Role" },
  {
    accessorKey: "status",
    header: "Status",
    cell: ({ row }) => (
      <span className={status === "active" ? "text-green-600" : "text-gray-500"}>
        {row.getValue("status")}
      </span>
    ),
  },
];

<Table
  data={users}
  columns={columns}
  enableVirtualization={false}
/>`}
    >
      <div className="w-full">
        <Table
          data={sampleUsers}
          columns={columns}
          enableVirtualization={false}
          maxHeight="400px"
        />
      </div>
    </RenderComponentWithSnippet>
  );
}

export function SortableTable() {
  const columns: ColumnDef<User>[] = [
    {
      accessorKey: "name",
      header: "Name",
      enableSorting: true,
    },
    {
      accessorKey: "email",
      header: "Email",
      enableSorting: true,
    },
    {
      accessorKey: "role",
      header: "Role",
      enableSorting: true,
    },
    {
      accessorKey: "lastSeen",
      header: "Last Seen",
      enableSorting: true,
      cell: ({ row }) => new Date(row.getValue("lastSeen") as Date).toLocaleDateString(),
    },
  ];

  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<Table
  data={users}
  columns={columns}
  enableSorting={true}
  enableVirtualization={false}
/>`}
    >
      <div className="w-full">
        <Table
          data={sampleUsers}
          columns={columns}
          enableSorting={true}
          enableVirtualization={false}
          maxHeight="400px"
        />
      </div>
    </RenderComponentWithSnippet>
  );
}

export function SearchableTable() {
  const columns: ColumnDef<User>[] = [
    { accessorKey: "name", header: "Name" },
    { accessorKey: "email", header: "Email" },
    { accessorKey: "role", header: "Role" },
    { accessorKey: "status", header: "Status" },
  ];

  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<Table
  data={users}
  columns={columns}
  enableGlobalFilter={true}
  enableColumnVisibility={true}
  enableVirtualization={false}
/>`}
    >
      <div className="w-full">
        <Table
          data={sampleUsers}
          columns={columns}
          enableGlobalFilter={true}
          enableColumnVisibility={true}
          enableVirtualization={false}
          maxHeight="400px"
        />
      </div>
    </RenderComponentWithSnippet>
  );
}

export function VirtualizedTable() {
  const columns: ColumnDef<ApiKey>[] = [
    {
      accessorKey: "name",
      header: "Name",
      size: 150,
    },
    {
      accessorKey: "prefix",
      header: "Key Prefix",
      size: 150,
      cell: ({ row }) => (
        <code className="text-xs bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded">
          {row.getValue("prefix") as string}
        </code>
      ),
    },
    {
      accessorKey: "requests",
      header: "Requests",
      size: 120,
      cell: ({ row }) => (row.getValue("requests") as number).toLocaleString(),
    },
    {
      accessorKey: "rateLimit",
      header: "Rate Limit",
      size: 120,
    },
    {
      accessorKey: "created",
      header: "Created",
      size: 150,
      cell: ({ row }) => new Date(row.getValue("created") as Date).toLocaleDateString(),
    },
  ];

  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<Table
  data={apiKeys} // 50+ items
  columns={columns}
  enableVirtualization={true}
  enableSorting={true}
  estimateSize={52}
  overscan={5}
  maxHeight="500px"
/>`}
    >
      <div className="w-full">
        <Table
          data={sampleApiKeys}
          columns={columns}
          enableVirtualization={true}
          enableSorting={true}
          estimateSize={52}
          overscan={5}
          maxHeight="500px"
        />
      </div>
    </RenderComponentWithSnippet>
  );
}

export function SelectableTable() {
  const columns: ColumnDef<User>[] = [
    {
      id: "select",
      header: ({ table }) => (
        <input
          type="checkbox"
          checked={table.getIsAllRowsSelected()}
          onChange={table.getToggleAllRowsSelectedHandler()}
          className="rounded border-gray-300"
        />
      ),
      cell: ({ row }) => (
        <input
          type="checkbox"
          checked={row.getIsSelected()}
          onChange={row.getToggleSelectedHandler()}
          className="rounded border-gray-300"
        />
      ),
      size: 50,
    },
    { accessorKey: "name", header: "Name" },
    { accessorKey: "email", header: "Email" },
    { accessorKey: "role", header: "Role" },
  ];

  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<Table
  data={users}
  columns={columnsWithCheckbox}
  enableRowSelection={true}
  enableMultiRowSelection={true}
  enableVirtualization={false}
/>`}
    >
      <div className="w-full">
        <Table
          data={sampleUsers}
          columns={columns}
          enableRowSelection={true}
          enableMultiRowSelection={true}
          enableVirtualization={false}
          maxHeight="400px"
        />
      </div>
    </RenderComponentWithSnippet>
  );
}

export function ResizableTable() {
  const columns: ColumnDef<User>[] = [
    {
      accessorKey: "name",
      header: "Name",
      size: 200,
      minSize: 100,
      maxSize: 400,
    },
    {
      accessorKey: "email",
      header: "Email",
      size: 250,
      minSize: 150,
      maxSize: 500,
    },
    {
      accessorKey: "role",
      header: "Role",
      size: 120,
      minSize: 80,
      maxSize: 200,
    },
  ];

  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<Table
  data={users}
  columns={columns}
  enableColumnResizing={true}
  persistenceKey="resizable-table"
  persistColumnSizing={true}
  enableVirtualization={false}
/>`}
    >
      <div className="w-full">
        <Table
          data={sampleUsers}
          columns={columns}
          enableColumnResizing={true}
          persistenceKey="resizable-table-demo"
          persistColumnSizing={true}
          enableVirtualization={false}
          maxHeight="400px"
        />
      </div>
    </RenderComponentWithSnippet>
  );
}

export function ExportableTable() {
  const columns: ColumnDef<User>[] = [
    { accessorKey: "name", header: "Name" },
    { accessorKey: "email", header: "Email" },
    { accessorKey: "role", header: "Role" },
    { accessorKey: "status", header: "Status" },
  ];

  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<Table
  data={users}
  columns={columns}
  enableExport={true}
  exportFilename="users-export"
  exportFormats={["csv", "json"]}
  enableVirtualization={false}
/>`}
    >
      <div className="w-full">
        <Table
          data={sampleUsers}
          columns={columns}
          enableExport={true}
          exportFilename="users-export"
          exportFormats={["csv", "json"]}
          enableVirtualization={false}
          maxHeight="400px"
        />
      </div>
    </RenderComponentWithSnippet>
  );
}

export function FullFeaturedTable() {
  const columns: ColumnDef<ApiKey>[] = [
    {
      accessorKey: "name",
      header: "Name",
      size: 150,
      enableSorting: true,
    },
    {
      accessorKey: "prefix",
      header: "Key Prefix",
      size: 150,
      cell: ({ row }) => (
        <code className="text-xs bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded font-mono">
          {row.getValue("prefix") as string}
        </code>
      ),
    },
    {
      accessorKey: "requests",
      header: "Requests",
      size: 120,
      enableSorting: true,
      cell: ({ row }) => (row.getValue("requests") as number).toLocaleString(),
    },
    {
      accessorKey: "rateLimit",
      header: "Rate Limit",
      size: 120,
    },
    {
      accessorKey: "created",
      header: "Created",
      size: 150,
      enableSorting: true,
      cell: ({ row }) => new Date(row.getValue("created") as Date).toLocaleDateString(),
    },
  ];

  return (
    <RenderComponentWithSnippet
      customCodeSnippet={`<Table
  data={apiKeys}
  columns={columns}
  enableVirtualization={true}
  enableSorting={true}
  enableGlobalFilter={true}
  enableColumnVisibility={true}
  enableColumnResizing={true}
  enableExport={true}
  exportFormats={["csv", "json"]}
  persistenceKey="full-featured-table"
  persistColumnSizing={true}
  persistColumnVisibility={true}
  maxHeight="600px"
/>`}
    >
      <div className="w-full">
        <Table
          data={sampleApiKeys}
          columns={columns}
          enableVirtualization={true}
          enableSorting={true}
          enableGlobalFilter={true}
          enableColumnVisibility={true}
          enableColumnResizing={true}
          enableExport={true}
          exportFormats={["csv", "json"]}
          persistenceKey="full-featured-table-demo"
          persistColumnSizing={true}
          persistColumnVisibility={true}
          maxHeight="600px"
        />
      </div>
    </RenderComponentWithSnippet>
  );
}
