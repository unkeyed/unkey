import { useCallback, useMemo, useRef } from "react";
import { LayoutEngine } from "../../layout-engine";
import type { TreeLayoutProps, TreeNode } from "../../types";
import type { DeploymentNode, NodeMetadata } from "../nodes/types";
import { TreeConnectionLine } from "./tree-connection-line";
import { TreeElementNode } from "./tree-element-node";

type NodeSize = { width: number; height: number };
/**
 * Since our nodes are custom-made, we can optimize layout through static heights and widths.
 * If things change over time, we can either update this list or create a ResizeObserver to track changes dynamically.
 */
const NODE_SIZES: Record<NodeMetadata["type"], NodeSize> = {
  origin: { width: 70, height: 20 },
  region: { width: 282, height: 100 },
  instance: { width: 282, height: 100 },
};

/**
 * Vertical tree layout component (top to bottom).
 * Uses fixed node dimensions for immediate layout calculation.
 */
export function TreeLayout<T extends TreeNode>({
  data,
  nodeSpacing = { x: 50, y: 50 },
  renderNode,
  renderConnection,
  onNodeClick,
}: TreeLayoutProps<T>) {
  const containerRef = useRef<SVGGElement>(null);

  const layoutEngine = useMemo(
    () => new LayoutEngine<T>({ spacing: nodeSpacing, direction: "vertical" }),
    [nodeSpacing],
  );

  const parentMap = useMemo(() => {
    const map = new Map<string, T>();
    const buildMap = (node: T) => {
      if (node.children) {
        for (const child of node.children) {
          map.set(child.id, node);
          buildMap(child as T);
        }
      }
    };
    buildMap(data);
    return map;
  }, [data]);

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

  const handleClick = useCallback(
    (e: React.MouseEvent<SVGGElement>) => {
      if (!onNodeClick) {
        return;
      }
      let target = e.target as HTMLElement | SVGElement;
      while (target && target !== e.currentTarget) {
        const nodeId = target.getAttribute("data-node-id");
        if (nodeId) {
          const node = allNodes.find((n) => n.id === nodeId);
          if (node) {
            onNodeClick(node);
          }
          break;
        }
        target = target.parentElement as HTMLElement | SVGElement;
      }
    },
    [onNodeClick, allNodes],
  );

  // Use node-specific size or fallback to default
  // Set dimensions based on node type
  allNodes.forEach((node) => {
    // Type assertion since we know DeploymentNode has metadata
    const deploymentNode = node as unknown as DeploymentNode;
    const size = NODE_SIZES[deploymentNode.metadata.type];
    layoutEngine.setNodeDimension(node.id, size);
  });

  const layout = useMemo(() => {
    return layoutEngine.calculate(data);
  }, [data, layoutEngine]);

  return (
    // biome-ignore lint/a11y/useKeyWithClickEvents: This is required for event bubbling
    <g ref={containerRef} onClick={handleClick}>
      {layout.connections.map((conn) =>
        renderConnection ? (
          renderConnection(conn.path, conn.parent, conn.child)
        ) : (
          <TreeConnectionLine key={`${conn.parent.id}-${conn.child.id}`} path={conn.path} />
        ),
      )}

      {layout.nodes.map((positioned) => {
        const parent = parentMap.get(positioned.node.id);
        return (
          <TreeElementNode
            key={positioned.node.id}
            id={positioned.node.id}
            position={positioned.position}
          >
            {renderNode(positioned.node, positioned.position, parent)}
          </TreeElementNode>
        );
      })}
    </g>
  );
}
