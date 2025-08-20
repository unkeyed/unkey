export const ROOT_KEY_CONSTANTS = {
  DEFAULT_LIMIT: 10,
  UNNAMED_KEY: "Unnamed",
  API_URL: "https://api.unkey.com",
  EXPAND_COUNT: 3,
  WORKSPACE: "workspace",
} as const;

export const ROOT_KEY_MESSAGES = {
  SUCCESS: {
    ROOT_KEY_CREATED: "Root key created",
    ROOT_KEY_UPDATED_NAME: "Root key name updated",
    ROOT_KEY_GENERATED: "You've successfully generated a new Root key.",
    ROOT_KEY_UPDATED_PERMISSIONS:
      "Root key permissions updated successfully, New permissions count: ",
  },
  ERROR: {
    ACTION_FAILED: "Action Failed",
    ACTION_FAILED_DESCRIPTION: "An unexpected error occurred. Please try again.",
    NO_PENDING_ACTION: "No pending action when confirming close",
  },
  WARNING: {
    WONT_SEE_AGAIN: "You won't see this secret key again!",
    COPY_BEFORE_CLOSING:
      "Make sure to copy your secret key before closing. It cannot be retrieved later.",
  },
  UI: {
    ROOT_KEY: "Root Key",
    CLOSE_ANYWAY: "Close anyway",
    DISMISS: "Dismiss",
    LOADING: "Loading...",
    SELECT_PERMISSIONS: "Select Permissions...",
    NO_RESULTS: "No results found",
    FROM_APIS: "From APIs",
    ALL_SET: "All set! You can now create another key or explore the docs to learn more",
    LEARN_MORE: "Learn more",
    COPY_SAVE_KEY: "Copy and save this key secret as it won't be shown again.",
    TRY_IT_OUT: "Try It Out",
    KEY_DETAILS: "Key Details",
    KEY_SECRET: "Key Secret",
    SEE_KEY_DETAILS: "See key details",
    CREATE_ROOT_KEY: "Create root key",
    UPDATE_ROOT_KEY: "Update root key",
    NEW_ROOT_KEY: "New root key",
    NEW_ROOT_KEY_SUBTITLE: "Define a new root key and assign permissions",
    LOAD_MORE: "Load More",
    SEARCH_PERMISSIONS: "Search permissions",
    CLEAR_SEARCH: "Clear search",
  },
  PLACEHOLDERS: {
    KEY_NAME: "e.g. Vercel Production",
  },
  DESCRIPTIONS: {
    KEY_NAME_LABEL: "Key Name",
    KEY_NAME_DESCRIPTION: "Give your key a name, this is not customer facing.",
    PERMISSIONS: "Permissions",
    WORKSPACE: "All workspace permissions",
    API: "All permissions for",
    IMMEDIATE_CREATE: "This root key will be created immediately",
    IMMEDIATE_UPDATE: "This root key will be updated immediately",
  },
} as const;
