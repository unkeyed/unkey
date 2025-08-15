# Root Key Components

This directory contains the refactored root key management components for the Unkey dashboard.

## Architecture Overview

The components have been refactored to follow a clean architecture pattern with:

- **Custom Hooks**: Business logic extracted into reusable hooks
- **Utility Functions**: Shared logic centralized in utility files
- **Constants**: Centralized configuration and messages
- **Smaller Components**: Components broken down into focused, single-responsibility pieces

## Directory Structure

```
root-key/
├── components/           # UI components
│   ├── expandable-category.tsx
│   ├── highlighted-text.tsx
│   ├── permission-badge-list.tsx
│   ├── permission-list.tsx
│   ├── permission-sheet.tsx
│   ├── permission-toggle.tsx
│   ├── search-input.tsx
│   └── search-permissions.tsx
├── hooks/               # Custom hooks for business logic
│   ├── use-permissions.ts
│   ├── use-permission-sheet.ts
│   ├── use-root-key-dialog.ts
│   └── use-root-key-success.ts
├── utils/               # Utility functions
│   └── permissions.ts
├── constants.ts         # Shared constants and messages
├── create-rootkey-button.tsx
├── root-key-dialog.tsx
├── root-key-success.tsx
└── README.md
```

## Key Improvements

### 1. Custom Hooks
- **`usePermissions`**: Manages permission state and logic
- **`usePermissionSheet`**: Handles permission sheet state and search
- **`useRootKeyDialog`**: Manages root key dialog state and API calls
- **`useRootKeySuccess`**: Handles success dialog state and navigation

### 2. Utility Functions
- **`permissions.ts`**: Centralized permission management utilities
- **`constants.ts`**: Shared constants and user-facing messages

### 3. Component Refactoring
- **Smaller, focused components**: Each component has a single responsibility
- **Better separation of concerns**: UI logic separated from business logic
- **Improved reusability**: Components are more modular and reusable
- **Enhanced type safety**: Better TypeScript types throughout

### 4. Code Organization
- **Reduced duplication**: Common logic extracted to utilities and hooks
- **Consistent patterns**: Similar components follow the same patterns
- **Better maintainability**: Easier to understand and modify individual pieces

## Usage Examples

### Creating a Root Key
```tsx
import { CreateRootKeyButton } from "./create-rootkey-button";

<CreateRootKeyButton />
```

### Using Permission Management
```tsx
import { usePermissions } from "./hooks/use-permissions";

const { state, handlePermissionToggle } = usePermissions({
  type: "workspace",
  selected: permissions,
  onPermissionChange: setPermissions,
});
```

### Using Constants
```tsx
import { ROOT_KEY_MESSAGES } from "./constants";

<Button>{ROOT_KEY_MESSAGES.UI.CREATE_ROOT_KEY}</Button>
```

## Benefits

1. **Maintainability**: Easier to understand and modify individual components
2. **Reusability**: Hooks and utilities can be reused across different parts of the app
3. **Testability**: Business logic in hooks is easier to test in isolation
4. **Type Safety**: Better TypeScript support with proper types
5. **Performance**: Optimized re-renders with proper dependency management
6. **Consistency**: Standardized patterns across all components

## Migration Notes

The refactoring maintains backward compatibility while providing a cleaner architecture. All existing functionality is preserved, but the code is now more organized and maintainable. 