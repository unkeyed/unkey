"use client";

/**
 * Collapsible resource tree. Collection segments render as group headers,
 * id segments render as selectable rows with a human label and dimmed id.
 * Top-level collections and resource nodes collapse; nested collection headers
 * always show once their parent resource is expanded.
 */

import { ChevronDown, ChevronRight } from "@unkey/icons";
import { cn } from "@unkey/ui";
import { useState } from "react";
import { type TreeNode, childSummary, collectionCount, collectionName, countLabel } from "./model";

export function ResourceTree({
  nodes,
  selectedPath,
  onSelect,
}: {
  nodes: TreeNode[];
  selectedPath: string | null;
  onSelect: (path: string) => void;
}) {
  const [expanded, setExpanded] = useState<ReadonlySet<string>>(new Set(["keyspaces"]));

  const toggle = (path: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
  };

  if (nodes.length === 0) {
    return <p className="px-2 py-3 text-sm text-gray-10">No resources in this workspace yet.</p>;
  }

  return (
    <div className="flex flex-col gap-0.5" role="tree" aria-label="Workspace resources">
      {nodes.map((node) => (
        <NodeRow
          key={node.path}
          node={node}
          depth={0}
          expanded={expanded}
          toggle={toggle}
          selectedPath={selectedPath}
          onSelect={onSelect}
        />
      ))}
    </div>
  );
}

interface RowProps {
  node: TreeNode;
  depth: number;
  expanded: ReadonlySet<string>;
  toggle: (path: string) => void;
  selectedPath: string | null;
  onSelect: (path: string) => void;
}

function NodeRow(props: RowProps) {
  return props.node.resource === null ? <CollectionRow {...props} /> : <ResourceRow {...props} />;
}

function CollectionRow({ node, depth, expanded, toggle, selectedPath, onSelect }: RowProps) {
  // Only top-level collections collapse; nested ones ("keys" under a keyspace)
  // are gated by their parent resource's chevron instead.
  const collapsible = depth === 0;
  const isOpen = !collapsible || expanded.has(node.path);
  const count = collectionCount(node);

  const header = (
    <>
      {collapsible && (
        <span className="flex h-5 w-4 shrink-0 items-center justify-center text-gray-9">
          {isOpen ? <ChevronDown iconSize="sm-regular" /> : <ChevronRight iconSize="sm-regular" />}
        </span>
      )}
      <span className="text-[11px] font-medium uppercase tracking-wide text-gray-10">
        {collectionName(node.segment)}
      </span>
      {count > 0 && (
        <span className="text-[11px] text-gray-9">{countLabel(node.segment, count)}</span>
      )}
    </>
  );

  return (
    // biome-ignore lint/a11y/useSemanticElements: tree-branch grouping; fieldset/optgroup are form constructs and do not apply here
    <div role="group">
      {collapsible ? (
        <button
          type="button"
          onClick={() => toggle(node.path)}
          aria-expanded={isOpen}
          className="flex w-full items-center gap-1.5 rounded-md px-1 py-1.5 text-left transition-colors hover:bg-grayA-2"
        >
          {header}
        </button>
      ) : (
        <div className="flex items-center gap-1.5 px-1 pb-0.5 pt-1.5">{header}</div>
      )}
      {isOpen && (
        <div className="ml-2 flex flex-col gap-0.5 border-l border-grayA-4 pl-1">
          {node.children.map((child) => (
            <NodeRow
              key={child.path}
              node={child}
              depth={depth + 1}
              expanded={expanded}
              toggle={toggle}
              selectedPath={selectedPath}
              onSelect={onSelect}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function ResourceRow({ node, depth, expanded, toggle, selectedPath, onSelect }: RowProps) {
  const resource = node.resource;
  if (!resource) {
    return null;
  }

  const isSelected = selectedPath === node.path;
  const hasChildren = node.children.length > 0;
  const isOpen = hasChildren && expanded.has(node.path);
  const summary = childSummary(node);

  return (
    <div>
      <div
        className={cn(
          "flex items-center rounded-md transition-colors",
          isSelected ? "bg-grayA-3" : "hover:bg-grayA-2",
        )}
      >
        {hasChildren ? (
          <button
            type="button"
            onClick={() => toggle(node.path)}
            aria-label={isOpen ? `Collapse ${resource.label}` : `Expand ${resource.label}`}
            aria-expanded={isOpen}
            className="flex h-7 w-5 shrink-0 items-center justify-center text-gray-9 transition-colors hover:text-gray-12"
          >
            {isOpen ? (
              <ChevronDown iconSize="sm-regular" />
            ) : (
              <ChevronRight iconSize="sm-regular" />
            )}
          </button>
        ) : (
          <span className="w-5 shrink-0" />
        )}
        <button
          type="button"
          onClick={() => onSelect(node.path)}
          aria-pressed={isSelected}
          className="flex min-w-0 flex-1 items-baseline gap-2 py-1.5 pr-2 text-left"
        >
          <span
            className={cn(
              "truncate text-[13px]",
              isSelected ? "font-medium text-accent-12" : "text-gray-12",
            )}
          >
            {resource.label}
          </span>
          {resource.label !== node.segment && (
            <span className="truncate font-mono text-[11px] text-gray-9">{node.segment}</span>
          )}
          {summary && !isOpen && (
            <span className="ml-auto whitespace-nowrap pl-2 text-[11px] text-gray-9">
              {summary}
            </span>
          )}
        </button>
      </div>
      {isOpen && (
        <div className="ml-2 flex flex-col gap-0.5 border-l border-grayA-4 pl-1">
          {node.children.map((child) => (
            <NodeRow
              key={child.path}
              node={child}
              depth={depth + 1}
              expanded={expanded}
              toggle={toggle}
              selectedPath={selectedPath}
              onSelect={onSelect}
            />
          ))}
        </div>
      )}
    </div>
  );
}
