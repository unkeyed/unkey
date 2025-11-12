import type { TreeNode, Point, PositionedNode } from "./types";

/**
 * Dimensions of a rendered node in pixels
 */
type NodeDimensions = {
  width: number;
  height: number;
};

/**
 * Configuration for tree layout calculation
 */
type LayoutConfig = {
  /** Space between adjacent nodes */
  spacing: { x: number; y: number };
  /** Tree growth direction */
};

/**
 * Complete layout calculation result containing positioned nodes and their connections
 */
type LayoutResult<T extends TreeNode> = {
  nodes: PositionedNode<T>[];
  connections: Array<{
    from: Point;
    to: Point;
    parent: T;
    child: T;
    waypoints?: Point[];
  }>;
};

/**
 * Pure layout calculation engine for tree structures.
 * Requires all node dimensions before calculating positions.
 * Throws immediately on missing data or invalid state.
 */
export class LayoutEngine<T extends TreeNode> {
  private config: LayoutConfig;
  private dimensions: Map<string, NodeDimensions>;

  constructor(config: LayoutConfig) {
    this.config = config;
    this.dimensions = new Map();
  }

  /**
   * Register measured dimensions for a node.
   * Must be called for all nodes before calculate().
   */
  setNodeDimension(id: string, dimensions: NodeDimensions): void {
    this.dimensions.set(id, dimensions);
  }

  /**
   * Check if all nodes in the tree have registered dimensions.
   */
  hasAllDimensions(root: T): boolean {
    const allNodes = this.flattenTree(root);
    return allNodes.every((node) => this.dimensions.has(node.id));
  }

  /**
   * Calculate absolute positions for all nodes and their connections.
   * @throws Error if any node is missing dimensions
   */
  calculate(root: T): LayoutResult<T> {
    if (!this.hasAllDimensions(root)) {
      throw new Error(
        `Cannot calculate layout: missing dimensions for some nodes. Have ${this.dimensions.size} dimensions.`
      );
    }

    const positioned = this.layoutVertical(root);

    const connections = this.buildConnections(positioned);

    return { nodes: positioned, connections };
  }

  clear(): void {
    this.dimensions.clear();
  }

  /**
   * Flatten tree into linear array using depth-first traversal
   */
  private flattenTree(root: T): T[] {
    const result: T[] = [root];
    if (root.children) {
      root.children.forEach((child) => {
        result.push(...this.flattenTree(child as T));
      });
    }
    return result;
  }

  /**
   * Group nodes by their depth level using breadth-first traversal.
   * Level 0 is root, level 1 is root's children, etc.
   */
  private buildLevelGroups(root: T): Map<number, T[]> {
    const levelGroups: Map<number, T[]> = new Map();
    const queue: Array<{ node: T; level: number }> = [{ node: root, level: 0 }];

    while (queue.length > 0) {
      const nodeAndLevel = queue.shift();
      if (!nodeAndLevel) {
        throw new Error("Queue shift returned undefined");
      }
      const { node, level } = nodeAndLevel;

      if (!levelGroups.has(level)) {
        levelGroups.set(level, []);
      }
      const currentLevel = levelGroups.get(level);
      if (!currentLevel) {
        throw new Error(`Level ${level} map entry is undefined`);
      }

      currentLevel.push(node);

      if (node.children) {
        node.children.forEach((child) => {
          queue.push({ node: child as T, level: level + 1 });
        });
      }
    }

    return levelGroups;
  }

  /**
   * Calculate positions for vertical tree layout (top to bottom).
   * Children are distributed evenly across parent's horizontal span.
   * Subtrees are spaced to prevent overlap between sibling subtrees.
   */
  private layoutVertical(root: T): PositionedNode<T>[] {
    const positioned: PositionedNode<T>[] = [];

    // First pass: calculate subtree widths for all nodes
    const subtreeWidths = this.calculateSubtreeWidths(root);

    // Second pass: position nodes level by level
    const levelGroups = this.buildLevelGroups(root);
    let currentY = 0;

    levelGroups.forEach((nodes, level) => {
      // Find tallest node in this level to determine vertical spacing.
      // All nodes in a level share the same Y coordinate, but we need
      // to reserve enough vertical space for the tallest one.
      const maxHeight = Math.max(
        ...nodes.map((n) => {
          const dim = this.dimensions.get(n.id);
          if (!dim) {
            throw new Error(`Missing dimensions for node ${n.id}`);
          }
          return dim.height;
        })
      );

      if (level === 0) {
        // Root node: center at origin
        const rootDim = this.dimensions.get(nodes[0].id);
        if (!rootDim) {
          throw new Error("Missing dimensions for root node");
        }

        positioned.push({
          node: nodes[0],
          position: { x: 0, y: currentY + maxHeight / 2 },
          level,
        });
      } else {
        // Position nodes at this level based on their parent and siblings
        nodes.forEach((node) => {
          const parent = this.findParent(root, node.id);
          if (!parent) {
            throw new Error(`Cannot find parent for node ${node.id}`);
          }

          const parentPos = positioned.find((p) => p.node.id === parent.id);
          if (!parentPos) {
            throw new Error(`Parent position not found for ${node.id}`);
          }

          const siblings = parent.children || [];
          const siblingIndex = siblings.findIndex((s) => s.id === node.id);

          const dim = this.dimensions.get(node.id);
          if (!dim) {
            throw new Error(`Missing dimensions for node ${node.id}`);
          }

          // Calculate position using subtree widths to prevent overlap
          const x = this.calculateChildXPosition(
            parentPos.position.x,
            siblingIndex,
            siblings.map((s) => {
              const subtreeW = subtreeWidths.get(s.id);
              if (!subtreeW) {
                throw new Error(
                  `Cannot find the subtree width of node ${node.id}`
                );
              }
              return subtreeW;
            })
          );

          positioned.push({
            node,
            position: { x, y: currentY + maxHeight / 2 },
            level,
          });
        });
      }

      currentY += maxHeight + this.config.spacing.y;
    });

    return positioned;
  }

  /**
   * Calculate the horizontal width required for each node's entire subtree.
   * Width includes the node itself plus all descendants laid out horizontally.
   */
  private calculateSubtreeWidths(root: T): Map<string, number> {
    const widths = new Map<string, number>();

    const calculate = (node: T): number => {
      const nodeDim = this.dimensions.get(node.id);
      if (!nodeDim) {
        throw new Error(`Missing dimensions for node ${node.id}`);
      }

      if (!node.children || node.children.length === 0) {
        // Leaf node: width is just the node itself
        widths.set(node.id, nodeDim.width);
        return nodeDim.width;
      }

      // Calculate width needed for all children's subtrees
      const childSubtreeWidths = node.children.map((child) =>
        calculate(child as T)
      );
      const totalChildWidth = childSubtreeWidths.reduce((sum, w) => sum + w, 0);
      const childSpacing = (node.children.length - 1) * this.config.spacing.x;
      const childrenWidth = totalChildWidth + childSpacing;

      // Subtree width is the maximum of:
      // 1. This node's width
      // 2. Total width of children subtrees
      const subtreeWidth = Math.max(nodeDim.width, childrenWidth);
      widths.set(node.id, subtreeWidth);

      return subtreeWidth;
    };

    calculate(root);
    return widths;
  }

  /**
   * Calculate X position for a child node based on subtree widths.
   * Uses each child's subtree width (not just node width) for spacing.
   */
  private calculateChildXPosition(
    parentX: number,
    childIndex: number,
    subtreeWidths: number[]
  ): number {
    const childCount = subtreeWidths.length;

    if (childCount === 1) {
      return parentX;
    }

    // Calculate total width needed for all subtrees + spacing
    const totalSubtreeWidth = subtreeWidths.reduce((sum, w) => sum + w, 0);
    const totalSpacing = (childCount - 1) * this.config.spacing.x;
    const totalWidth = totalSubtreeWidth + totalSpacing;

    // Start position (leftmost subtree's left edge)
    const startX = parentX - totalWidth / 2;

    // Calculate this child's X position (center of its subtree allocation)
    let x = startX;
    for (let i = 0; i < childIndex; i++) {
      x += subtreeWidths[i] + this.config.spacing.x;
    }
    x += subtreeWidths[childIndex] / 2;

    return x;
  }

  /**
   * Find parent node of a given node ID
   */
  private findParent(root: T, childId: string): T | null {
    if (root.children) {
      for (const child of root.children) {
        if (child.id === childId) {
          return root;
        }
        const found = this.findParent(child as T, childId);
        if (found) {
          return found;
        }
      }
    }
    return null;
  }
  /**
   * Build connection lines between parent and child nodes.
   * @throws Error if child node position not found
   */
  private buildConnections(
    positioned: PositionedNode<T>[]
  ): Array<{ from: Point; to: Point; parent: T; child: T }> {
    const connections: Array<{
      from: Point;
      to: Point;
      parent: T;
      child: T;
    }> = [];

    positioned.forEach((pos) => {
      if (pos.node.children) {
        pos.node.children.forEach((child) => {
          const childPos = positioned.find((p) => p.node.id === child.id);
          if (!childPos) {
            throw new Error(
              `Cannot find positioned node for child ${child.id}`
            );
          }

          const parentDim = this.dimensions.get(pos.node.id);
          const childDim = this.dimensions.get(child.id);

          if (!parentDim || !childDim) {
            throw new Error(
              `Missing dimensions for connection: ${pos.node.id} -> ${child.id}`
            );
          }

          connections.push({
            // From: bottom edge of parent (center X, bottom Y)
            from: {
              x: pos.position.x,
              y: pos.position.y + parentDim.height / 2,
            },
            // To: top edge of child (center X, top Y)
            to: {
              x: childPos.position.x,
              y: childPos.position.y - childDim.height / 2,
            },
            parent: pos.node,
            child: childPos.node as T,
          });
        });
      }
    });

    return connections;
  }

  // /**
  //  * Generate a formatted string representation of the calculated layout.
  //  * Useful for debugging and understanding the spatial arrangement.
  //  */
  // logLayout(result: LayoutResult<T>): string {
  //   const lines: string[] = [];

  //   // Header
  //   lines.push("=".repeat(80));
  //   lines.push("TREE LAYOUT CALCULATION RESULT");
  //   lines.push("=".repeat(80));
  //   lines.push("");

  //   // Group nodes by level
  //   const levelGroups = new Map<number, PositionedNode<T>[]>();
  //   result.nodes.forEach((node) => {
  //     if (!levelGroups.has(node.level)) {
  //       levelGroups.set(node.level, []);
  //     }
  //     levelGroups.get(node.level)!.push(node);
  //   });

  //   // Sort levels
  //   const sortedLevels = Array.from(levelGroups.keys()).sort((a, b) => a - b);

  //   // Print each level
  //   sortedLevels.forEach((level) => {
  //     const nodes = levelGroups.get(level)!;

  //     lines.push(`LEVEL ${level}`);
  //     lines.push("-".repeat(80));

  //     // Sort nodes by X position for cleaner output
  //     nodes.sort((a, b) => a.position.x - b.position.x);

  //     nodes.forEach((positioned) => {
  //       const dim = this.dimensions.get(positioned.node.id);
  //       if (!dim)
  //         throw new Error(`Missing dimensions for node ${positioned.node.id}`);

  //       const nodeInfo = [
  //         `  [${positioned.node.id}]`,
  //         `label: "${positioned.node.label}"`,
  //         `pos: (${positioned.position.x.toFixed(
  //           1
  //         )}, ${positioned.position.y.toFixed(1)})`,
  //         `size: ${dim.width}×${dim.height}`,
  //         positioned.node.children
  //           ? `children: ${positioned.node.children.length}`
  //           : "leaf",
  //       ].join(" | ");

  //       lines.push(nodeInfo);
  //     });

  //     lines.push("");
  //   });

  //   // Connection summary
  //   lines.push("CONNECTIONS");
  //   lines.push("-".repeat(80));
  //   lines.push(`Total connections: ${result.connections.length}`);
  //   lines.push("");

  //   result.connections.forEach((conn) => {
  //     const hasWaypoints = conn.waypoints && conn.waypoints.length > 0;
  //     const routeType = hasWaypoints ? "orthogonal" : "direct";

  //     let connStr = `  ${conn.parent.id} → ${conn.child.id}`;
  //     connStr += ` [${routeType}]`;
  //     connStr += ` from:(${conn.from.x.toFixed(1)},${conn.from.y.toFixed(1)})`;
  //     connStr += ` to:(${conn.to.x.toFixed(1)},${conn.to.y.toFixed(1)})`;

  //     if (hasWaypoints) {
  //       connStr += ` via:[${conn
  //         .waypoints!.map((p) => `(${p.x.toFixed(1)},${p.y.toFixed(1)})`)
  //         .join(", ")}]`;
  //     }

  //     lines.push(connStr);
  //   });

  //   lines.push("");
  //   lines.push("=".repeat(80));

  //   return lines.join("\n");
  // }

  // /**
  //  * Log layout to console with formatted output.
  //  */
  // printLayout(result: LayoutResult<T>): void {
  //   console.log(this.logLayout(result));
  // }
}
