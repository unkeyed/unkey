import { useMemo, useState } from "react";
import type { TreeLayoutProps, TreeNode } from "../../types";
import { LayoutEngine } from "../../layout-engine";
import { TreeElementNode } from "./tree-element-node";
import { TreeConnectionLine } from "./tree-connection-line";

/**
 * Vertical tree layout component (top to bottom).
 * Operates in two phases:
 * 1. Measurement: Render all nodes at origin to get dimensions
 * 2. Layout: Calculate and render final positions with connections
 */
export function TreeLayout<T extends TreeNode>({
  data,
  nodeSpacing = { x: 50, y: 50 },
  renderNode,
  renderConnection,
}: TreeLayoutProps<T>) {
  const [nodeDimensions, setNodeDimensions] = useState<
    Map<string, { width: number; height: number }>
  >(new Map());

  const layoutEngine = useMemo(
    () => new LayoutEngine<T>({ spacing: nodeSpacing }),
    [nodeSpacing]
  );

  const allNodes = useMemo(() => {
    const flatten = (node: T): T[] => {
      const result: T[] = [node];
      if (node.children) {
        node.children.forEach((child) => {
          result.push(...flatten(child as T));
        });
      }
      return result;
    };
    return flatten(data);
  }, [data]);

  // Sync dimensions to engine synchronously before any checks
  nodeDimensions.forEach((dim, id) => {
    layoutEngine.setNodeDimension(id, dim);
  });

  const isLayoutReady = layoutEngine.hasAllDimensions(data);

  const layout = useMemo(() => {
    if (!isLayoutReady) {
      return null;
    }
    const result = layoutEngine.calculate(data);
    if (process.env.NODE_ENV === "development") {
      // layoutEngine.printLayout(result);
    }

    return result;
  }, [data, layoutEngine, isLayoutReady]);

  // Lock in first measurement for each node.
  // This prevents remeasurement after canvas scale changes,
  // which would corrupt the layout with scaled dimensions.
  const handleNodeMeasure = (
    id: string,
    width: number,
    height: number
  ): void => {
    setNodeDimensions((prev) => {
      const existing = prev.get(id);
      if (existing?.width === width && existing?.height === height) {
        return prev;
      }

      const next = new Map(prev);
      next.set(id, { width, height });
      return next;
    });
  };

  // Phase 1: Measurement - render all nodes at origin
  if (!layout) {
    return (
      <>
        {allNodes.map((node) => (
          <TreeElementNode
            key={node.id}
            id={node.id}
            position={{ x: 0, y: 0 }}
            onMeasure={handleNodeMeasure}
          >
            {renderNode(node, { x: 0, y: 0 })}
          </TreeElementNode>
        ))}
      </>
    );
  }

  // Phase 2: Final layout - render positioned nodes and connections
  return (
    <>
      {layout.connections.map((conn) =>
        renderConnection ? (
          renderConnection(conn.from, conn.to, conn.parent, conn.child)
        ) : (
          <TreeConnectionLine
            key={`${conn.parent.id}-${conn.child.id}`}
            from={conn.from}
            to={conn.to}
          />
        )
      )}

      {layout.nodes.map((positioned) => (
        <TreeElementNode
          key={positioned.node.id}
          id={positioned.node.id}
          position={positioned.position}
          onMeasure={handleNodeMeasure}
        >
          {renderNode(positioned.node, positioned.position)}
        </TreeElementNode>
      ))}
    </>
  );
}
