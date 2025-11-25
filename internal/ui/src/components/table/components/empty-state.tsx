import { DEFAULT_EMPTY_STATE_CONFIG, TABLE_CLASS_NAMES } from "../constants";
import type { EmptyStateConfig } from "../types";

interface EmptyStateProps {
  config?: EmptyStateConfig;
}

export function EmptyState({ config = DEFAULT_EMPTY_STATE_CONFIG }: EmptyStateProps) {
  const { title, description, icon, action } = config;

  return (
    <div
      className={`${TABLE_CLASS_NAMES.empty} flex flex-col items-center justify-center py-12 px-4 text-center`}
    >
      {icon && <div className="mb-4 text-gray-400 dark:text-gray-600">{icon}</div>}

      <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">
        {title || DEFAULT_EMPTY_STATE_CONFIG.title}
      </h3>

      {description && (
        <p className="text-sm text-gray-500 dark:text-gray-400 max-w-sm mb-6">{description}</p>
      )}

      {action && (
        <button
          type="button"
          onClick={action.onClick}
          className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-primary-600 hover:bg-primary-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
        >
          {action.label}
        </button>
      )}
    </div>
  );
}
