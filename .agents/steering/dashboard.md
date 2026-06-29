# Dashboard Development Guide

This guide covers patterns and conventions for developing Unkey's client-side applications, including the dashboard and other web apps in the monorepo.

## Technology Stack

- **Framework**: Next.js 16.1.5 with App Router
- **UI Library**: React 19.2.4
- **Language**: TypeScript 5.7.3
- **Styling**: Tailwind CSS 4.2.1 with Radix Colors
- **UI Components**: `@unkey/ui` package (shared component library) + Radix UI primitives
- **Icons**: `@unkey/icons` package
- **State Management**: 
  - TanStack Query 4.36.1 (React Query) for server state
  - tRPC 10.45.2 with React Query integration
- **Forms**: React Hook Form 7.55.0 with Zod 4.3.5 validation
- **Tables**: TanStack Table 8.16.0
- **Virtualization**: TanStack Virtual 3.10.9
- **Charts**: Recharts 3.7.0, @ant-design/plots 1.2.5
- **Animation**: Framer Motion 12.29.0
- **Testing**: Vitest 3.2.4 with Testing Library
- **Package Manager**: pnpm 11.5.0 with workspaces
- **Build System**: Turbo 2.4.3+ for monorepo orchestration
- **Code Quality**: Biome 1.9.4+ for formatting and linting
- **Error Tracking**: Sentry
- **Authentication**: WorkOS 7.47.0 (with MFA/passkeys support)

## Dashboard Features

The Unkey dashboard provides management interfaces for both Auth and Deploy products:

### Auth Management
- **Workspaces**: Organization-level settings, billing (two-product: Auth + Deploy), and security (MFA/passkeys)
- **Keyspaces**: Create and manage keyspaces (formerly "APIs") with key configuration
- **Keys**: View, create, and manage API keys with permissions
- **Identities**: External user/entity management
- **Permissions & Roles**: RBAC configuration with resource-level permissions
- **Rate Limits**: Configure rate limit namespaces and overrides (DataTable with sorting/pagination)
- **Analytics**: Real-time verification logs and usage metrics
- **Audit Logs**: Security event history

### Deploy Management
- **Projects**: Organize related applications (gated by deploy plan)
- **Apps**: Manage containerized applications
- **Environments**: Configure production, staging, development environments
- **Deployments**: View deployment history, promote, rollback
- **Environment Variables**: Manage secrets and configuration per environment
- **Custom Domains**: Add and verify custom domains with automatic TLS
- **Regional Settings**: Configure replica counts per region
- **Build Settings**: Dockerfile path or automatic (Railpack) builds
- **Sentinel Logs**: Request-level logs for deployment proxy (DataTable with sorting/pagination)

## Feature-Based Architecture

Each feature should be organized as a self-contained module. Keep all related code within the feature directory.

### Directory Structure

```
feature-name/
├── components/              # Feature-specific React components
│   ├── component-name/      # Complex components get their own directory
│   │   ├── index.tsx
│   │   └── sub-component.tsx
│   └── simple-component.tsx
├── hooks/                   # Custom hooks for the feature
│   ├── queries/             # API query hooks
│   │   ├── use-feature-list.ts
│   │   └── use-feature-details.ts
│   └── use-feature-logic.ts
├── actions/                 # Server actions and API calls
│   └── feature-actions.ts
├── types/                   # TypeScript types and interfaces
│   └── feature.ts
├── schemas/                 # Validation schemas
│   └── feature.ts
├── utils/                   # Helper functions
│   └── feature-helpers.ts
├── constants.ts             # Feature-specific constants
└── page.tsx                 # Main page component
```

## Key Principles

### 1. Feature Isolation
- Keep all related code within the feature directory
- Don't import feature-specific components into other features
- Use shared components from `/components` or `@unkey/ui` for common UI elements

### 2. Component Organization
- Simple components can be single files (don't over-engineer - follow common sense)
- Complex components should have their own directory with `index.tsx`
- Keep component-specific styles, tests, and utilities close to the component

### 3. Code Colocation
- Place related code as close as possible to where it's used
- If a utility is only used by one component, keep it in the component's directory
- Only move code to shared directories when used across multiple features

## Contribution Guidelines

### Before Contributing
- **All contribution discussions MUST happen in public channels** (GitHub Issues/Discussions, Discord)
- **Do not send Direct Messages (DMs)** to team members about contributions
- For existing issues: Comment to express interest and wait for assignment
- For new ideas: Discuss first in public channels, get approval before coding

### Approval Process
- **Needs Prior Approval**: New features, refactoring, core functionality changes, UI/UX changes
- **Can Start Immediately**: Bug fixes, security improvements, documentation updates, typo corrections
- Always create an issue before starting development
- Reference the issue in your PR using `fixes #XXX` or `refs #XXX`

### House Rules
- Check existing issues and PRs before submitting new ones
- All new issues get a `needs-approval` label automatically
- Wait for core team member approval before beginning work on new features

## Page Structure Pattern

```typescript
import { Navbar } from "@/components/navbar"; // Global shared component
import { PageContent } from "@/components/page-content";
import { FeatureComponent } from "./components/feature-component";

export default function FeaturePage() {
  // Server-side data fetching happens here
  const data = await fetchData();
  
  return (
    <div>
      <Navbar>{/* Navigation content */}</Navbar>
      <PageContent>
        {/* Entry to our actual component (usually client-side) */}
        <FeatureComponent data={data} />
      </PageContent>
    </div>
  );
}
```

## File Naming Conventions

- Use **kebab-case** for directory and file names
- The directory structure provides context, so explicit suffixes are optional
- Common suffix patterns (when needed for clarity):
  - `auth.schema.ts` or `auth-schema.ts` for validation schemas
  - `auth.type.ts` or `auth-types.ts` for type definitions
  - `.client.tsx` for client-specific components
  - `.server.ts` for server-only code
  - `.action.ts` for server actions

## Import/Export Conventions

- **Use absolute imports** for shared components: `@/components`
- **Never use default exports** unless absolutely necessary
- **Use relative imports** within a feature
- Export complex components through index files
- Avoid circular dependencies

## Shared Code Organization

Place code in root-level directories only if used across multiple features:

```
/components          # Shared React components
/hooks               # Shared custom hooks
/utils               # Shared utilities
/types               # Shared TypeScript types
/constants           # Global constants
```

## Design System Integration

### Using @unkey/ui Components

**Critical Setup**: Always import the CSS styles in your application's main layout:

```typescript
// In app/layout.tsx - THIS IS CRITICAL
import "@unkey/ui/css"; // Required for proper styling
```

Import components directly from the package:

```typescript
import { Button, FormInput, Badge } from "@unkey/ui";
```

### Tailwind Configuration

Ensure your Tailwind config includes UI package contents. Shared packages live in `web/internal/` (not `web/packages/`):

```javascript
module.exports = {
  content: [
    "./app/**/*.{js,ts,jsx,tsx}",
    "./components/**/*.{js,ts,jsx,tsx}",
    "../../internal/ui/src/**/*.tsx",
    "../../internal/icons/src/**/*.tsx",
  ],
  // ...
}
```

### Troubleshooting UI Components

**If components look unstyled:**
1. Verify you imported `@unkey/ui/css` in your layout
2. Check Tailwind config includes UI package content paths
3. Ensure no conflicting global styles

**Best Practices:**
- Import CSS only once at the root level
- Use component props and variants instead of direct styling
- Follow component documentation for proper usage

## Styling Guidelines

### Color System

Unkey uses Radix Colors with a 12-step scale. Dark mode is handled automatically.

**Color Scale Usage:**
- Steps 1-2: App background, subtle background
- Steps 3-5: UI element backgrounds (subtle, hover, active)
- Steps 6-8: Borders (subtle, default, hover)
- Steps 9-10: Solid backgrounds (default, hover)
- Steps 11-12: Text (low-contrast, high-contrast)

**Available Color Palettes:**
- `gray` - Neutral UI elements
- `success` - Positive states
- `info` - Informational states
- `warning` - Warning states
- `error` - Error states
- `feature` - Feature highlights
- `orange` - Secondary accent
- `accent` - Primary accent

**Alpha Variants:** Add `A` suffix for transparency (e.g., `grayA-6`)

### Using Colors

```tsx
// Background colors
<div className="bg-gray-3 hover:bg-gray-4" />

// Text colors
<p className="text-gray-12">High contrast text</p>
<p className="text-gray-11">Low contrast text</p>

// Borders
<div className="border border-gray-7 focus:border-gray-8" />

// Semantic colors
<Badge variant="success">Active</Badge>
<Badge variant="error">Failed</Badge>
```

**Color Guidelines:**
- Dark mode is automatic - avoid manual `dark:` classes unless necessary
- Use semantic colors for meaning (success-9, error-9, etc.)
- Follow the 12-step scale for consistent visual hierarchy

### Icons

Import icons from `@unkey/icons`:

```typescript
import { Key, Shield, Trash } from "@unkey/icons";

// Use with semantic colors
<Key className="text-gray-11" />
<Shield className="text-success-9" />
```

**Icon Guidelines:**
- Default size is appropriate for most cases
- Only customize color, avoid changing size unless necessary
- Use semantic colors for meaning (error-9, success-9, etc.)

## API Integration

### Response Structure

All Unkey API responses follow this structure:

```typescript
{
  meta: {
    requestId: string;
    timestamp?: string;
  },
  data: T;
  pagination?: {
    cursor: string;
    hasMore: boolean;
  }
}
```

### Error Handling

Always include the `requestId` when logging errors or seeking support:

```typescript
try {
  const response = await fetch('/api/endpoint');
  const { data, meta } = await response.json();
  // Use data
} catch (error) {
  console.error('Request failed', { requestId: meta.requestId });
}
```

## Testing

Run tests with:

```bash
# Run all tests
pnpm test

# Run tests for specific workspace
pnpm --filter @unkey/dashboard test

# Run tests with UI
pnpm test --ui
```

### Test Organization
- Colocate tests with the code they test
- Use descriptive test names
- Follow the Arrange-Act-Assert pattern
- Mock external dependencies

## Best Practices

### Component Design
- Keep components focused and single-purpose
- Use composition over inheritance
- Prefer controlled components for forms
- Handle loading and error states explicitly

### Performance
- Use React Server Components by default
- Add `"use client"` only when needed (interactivity, hooks, browser APIs, TanStack libraries, tRPC hooks)
- Lazy load heavy components
- Optimize images with Next.js Image component
- Use TanStack Query's built-in caching and deduplication via tRPC
- Implement pagination with TanStack Table for large datasets
- Use TanStack Virtual for long lists

### Accessibility
- Use semantic HTML elements
- Provide proper ARIA labels
- Ensure keyboard navigation works
- Test with screen readers
- Maintain proper color contrast (Radix Colors handles this)

### Type Safety
- Define explicit types for props and state
- Avoid `any` type
- Use TypeScript strict mode
- Define schemas for API responses and form validation

## Common Patterns

### Server Actions

```typescript
// actions/feature-actions.ts
"use server";

export async function createFeature(data: FeatureInput) {
  // Server-side logic
  return { success: true, data };
}
```

### Query Hooks (TanStack Query via tRPC)

```typescript
// hooks/queries/use-features.ts
import { trpc } from "@/lib/trpc";

export function useFeatures() {
  return trpc.features.list.useQuery();
}

// With mutations
export function useCreateFeature() {
  const utils = trpc.useUtils();
  
  return trpc.features.create.useMutation({
    onSuccess: () => {
      // Invalidate and refetch
      utils.features.list.invalidate();
    },
  });
}

// Direct TanStack Query usage (when not using tRPC)
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";

export function useDirectQuery() {
  return useQuery({
    queryKey: ['features'],
    queryFn: fetchFeatures,
  });
}
```

### Form Handling (React Hook Form + Zod)

```typescript
import { FormInput, Button } from "@unkey/ui";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";

const featureSchema = z.object({
  name: z.string().min(1, "Name is required"),
  description: z.string().optional(),
});

type FeatureFormData = z.infer<typeof featureSchema>;

export function FeatureForm() {
  const { register, handleSubmit, formState: { errors } } = useForm<FeatureFormData>({
    resolver: zodResolver(featureSchema),
  });

  const onSubmit = async (data: FeatureFormData) => {
    await createFeature(data);
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <FormInput
        label="Name"
        description="Enter a descriptive name"
        {...register("name")}
        error={errors.name?.message}
        required
      />
      <FormInput
        label="Description"
        {...register("description")}
        error={errors.description?.message}
      />
      <Button type="submit">Create</Button>
    </form>
  );
}
```

### tRPC Integration

```typescript
// hooks/queries/use-features.ts
import { trpc } from "@/lib/trpc";

export function useFeatures() {
  return trpc.features.list.useQuery();
}

export function useCreateFeature() {
  const utils = trpc.useUtils();
  
  return trpc.features.create.useMutation({
    onSuccess: () => {
      utils.features.list.invalidate();
    },
  });
}
```

### Data Tables (TanStack Table)

```typescript
import { useReactTable, getCoreRowModel, flexRender } from "@tanstack/react-table";

export function FeatureTable({ data }: { data: Feature[] }) {
  const table = useReactTable({
    data,
    columns: [
      {
        accessorKey: 'name',
        header: 'Name',
      },
      {
        accessorKey: 'status',
        header: 'Status',
      },
    ],
    getCoreRowModel: getCoreRowModel(),
  });

  return (
    <table>
      <thead>
        {table.getHeaderGroups().map(headerGroup => (
          <tr key={headerGroup.id}>
            {headerGroup.headers.map(header => (
              <th key={header.id}>
                {flexRender(header.column.columnDef.header, header.getContext())}
              </th>
            ))}
          </tr>
        ))}
      </thead>
      <tbody>
        {table.getRowModel().rows.map(row => (
          <tr key={row.id}>
            {row.getVisibleCells().map(cell => (
              <td key={cell.id}>
                {flexRender(cell.column.columnDef.cell, cell.getContext())}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
}
```

## tRPC Query Security

### Always scope queries to `ctx.workspace.id`

Every tRPC query that accepts user input (e.g., `projectId`, `apiId`, `keyId`) must include `ctx.workspace.id` in its `WHERE` clause or join condition — even if an outer query or join already checks workspace ownership. User-supplied IDs are untrusted; without explicit workspace scoping, a user could potentially access resources belonging to another workspace by guessing or enumerating IDs.

```typescript
// BAD: trusts that projectId alone is sufficient
.where(eq(frontlineRoutes.projectId, input.projectId))

// GOOD: always pair user input with workspace scoping
.where(
  and(
    eq(frontlineRoutes.projectId, input.projectId),
    eq(projects.workspaceId, ctx.workspace.id),
  ),
)
```

This applies to subqueries too. If a scalar subquery filters by a user-facing ID, join through the parent table to enforce workspace isolation:

```sql
-- BAD: no workspace check in subquery
SELECT fr.fully_qualified_domain_name
FROM frontline_routes fr
WHERE fr.project_id = projects.id

-- GOOD: join through projects to enforce workspace
SELECT fr.fully_qualified_domain_name
FROM frontline_routes fr
INNER JOIN projects p ON p.id = fr.project_id AND p.workspace_id = ?
WHERE fr.project_id = projects.id
```

## Navigation & Routing

### SSoT Route Constructors

All dashboard navigation URLs must use the type-safe route constructors in `lib/navigation/routes/`. This is enforced by a guard rail test (`no-handrolled-routes.test.ts`) that scans for manually constructed route strings.

```typescript
// Good: use route constructors
import { routes } from "@/lib/navigation/routes";
const url = routes.auth.keyspace(workspaceSlug, apiId);

// Bad: hand-rolled route strings
const url = `/${workspaceSlug}/apis/${apiId}`;
```

Route modules are organized by domain:
- `routes/auth.ts` - Keyspace and key routes
- `routes/authorization.ts` - Permissions and roles
- `routes/identities.ts` - Identity management
- `routes/settings.ts` - Workspace settings (including security)
- `routes/workspaces.ts` - Workspace-level routes
- `routes/audit.ts` - Audit log routes
- `routes/logs.ts` - Verification logs
- `routes/shared.ts` - Deploy routes (projects, apps, environments, deployments)

### Page Layout Primitives

Use `@unkey/ui` page composition components for consistent layouts:

```typescript
import { PageContainer, PageHeader, PageBody, SecondaryNav } from "@unkey/ui";

// Default width (centered, constrained)
<PageContainer>
  <PageHeader>
    <h1>Page Title</h1>
  </PageHeader>
  <PageBody>
    <Content />
  </PageBody>
</PageContainer>

// Full width (edge-to-edge body, e.g., for dense tables)
<PageContainer width="full">
  <PageHeader>...</PageHeader>
  <PageBody>...</PageBody>
</PageContainer>

// With secondary navigation rail
<PageContainer>
  <SecondaryNav items={[...]} />
  <PageBody>...</PageBody>
</PageContainer>
```

## Questions?

If you're unsure about where to place code or how to structure a feature, ask in Discord or in your pull request.
