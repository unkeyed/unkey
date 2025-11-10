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
   * Each level gets Y position based on max height in that level.
   * Nodes within level are centered horizontally.
   */
  private layoutVertical(root: T): PositionedNode<T>[] {
    const positioned: PositionedNode<T>[] = [];
    const levelGroups = this.buildLevelGroups(root);

    // Calculate Y positions
    const levelYPositions: number[] = [];
    let currentY = 0;

    levelGroups.forEach((nodes, level) => {
      const maxHeight = Math.max(
        ...nodes.map((n) => {
          const dim = this.dimensions.get(n.id);
          if (!dim) {
            throw new Error(`Missing dimensions for node ${n.id}`);
          }
          return dim.height;
        })
      );
      levelYPositions[level] = currentY;
      currentY += maxHeight + this.config.spacing.y;
    });

    // Calculate X positions
    levelGroups.forEach((nodes, level) => {
      const totalWidth = nodes.reduce((sum, n) => {
        const dim = this.dimensions.get(n.id);
        if (!dim) {
          throw new Error(`Missing dimensions for node ${n.id}`);
        }
        return sum + dim.width + this.config.spacing.x;
      }, -this.config.spacing.x);

      let currentX = -totalWidth / 2;

      nodes.forEach((node) => {
        const dim = this.dimensions.get(node.id);
        if (!dim) {
          throw new Error(`Missing dimensions for node ${node.id}`);
        }
        positioned.push({
          node,
          position: {
            x: currentX + dim.width / 2,
            y: levelYPositions[level],
          },
          level,
        });
        currentX += dim.width + this.config.spacing.x;
      });
    });

    return positioned;
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
          connections.push({
            from: pos.position,
            to: childPos.position,
            parent: pos.node,
            child: childPos.node as T,
          });
        });
      }
    });

    return connections;
  }
}
