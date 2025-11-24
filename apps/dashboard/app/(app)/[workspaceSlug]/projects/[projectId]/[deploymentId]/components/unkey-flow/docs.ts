/*
Flow of Layout Engine

Since layout engine is our backend, before we do anything, engine has to now the dimensions of the nodes and this is the responsibility of frontend.
In our case nodes are living in a tree, so we have to flatten them first, and then, pass the size of each node type to engine. Currently we predefine the dimensions of nodes.
In the future to make it dynamic we could render the nodes initially on our canvas, then simply query the DOM to get their actual sizes, but for now we know the node types and sizes beforehand there is no need for that.
If we accidentally call the `calculate` before setting the dimensions code will panic.

Then, we invoke the `calculate` to calculate the positioning of the nodes and their connection lines. First we call `buildNodeLayout` recursively.
For the initial node it is simple, since its level is `0` and position is `{x:0,y:0}` we directly push it into our `positioned: PositionedNode<T>[] = [];` array.
For the rest of the children nodes we'll run a depth first search, meaning, we'll first start with the first child and consume entire subtrees until we are done. Then, move on to next child and do the same.

Imagine we have ["internet", "us-east-1", "eu-central-1", ...gws]

                    INTERNET (0,0)
                          |
              +-----------+-----------+
              |                       |
         us-east-1              eu-central-1
              |                       |
          +------+                    +
          |      |                    |
        gw-1  gw-2                  gw-3

We would end up with this:
[
  level 0: INTERNET
  level 1: us-east-1
  level 2: gw-1
  level 2: gw-2
  level 1: eu-central-1
  level 2: gw-4
]

That's the easy part, but how are we going to ensure nodes don't overlap?
To ensure there is no overlap between nodes, each level has to be at least as big
as the subtree + node spacing. And, we have to do this recursively because we don't
know how many levels we will have. This is where `calculateSubtreeWidth` comes into play.
We also run the same logic for vertical stacking `calculateSubtreeHeight`.

Subtree width calculation example (bottom-up):
- gw-1, gw-2 : width = 282 (base case, no children, using node.width)
- us-east-1 subtree: width = 282 + 25 + 282 = 589
  (three gateways + two spacings between them)
- gw-4: width = 282 (base case, no children)
- eu-central-1 subtree: width = 282 (only one child, no spacing needed)

INTERNET subtree: width = 589 + 25 + 282 = 896
  (us-east-1 subtree + spacing + eu-central-1 subtree)



### Horizontal stacking logic

Now we know the width of the subtree, we can calculate the actual `X` positions of the nodes.
First we need to figure out where the first child's subtree starts, we want to center all children under the parent.
If parent is at X=0 and total width is `589` - meaning we have two children - thats why we divide it by 2.
Now we know where to place our first node, following nodes start where the previous node's subtree ended, plus spacing.
So, if our initial node starts from `-294.5` next node will start from `-294.5 + 282 + 25` and so on. And, finally since we only care about the
center of the nodes we divide it by 2 again and store that.


us-east-1 is at some position, let's say X=0 for this calculation
Children: [gw-1(width=282), gw-2(width=282)]
spacing.x = 25

Step 1: Total width
  282 + 282 + 25 = 589

Step 2: Start position (leftmost edge)
  startX = 0 - 589/2 = -294.5

Step 3 & 4: Calculate each child's center

  gw-1 (index=0):
    x = -294.5 + 0 = -294.5  -> No PRIOR sibling, so left edge is our start
    x += 282/2 = -153.5  ← gw-1's center

  gw-2 (index=1):
    x = -294.5 + (282 + 25) = -294.5 + 307 = 12.5   -> Left edge of the second node
    x += 282/2 = 153.5  ← gw-2's center


When a node has `direction="vertical"`, its children spread **horizontally** below it at the same Y position.

INTERNET (y=0, h=20) → us-east-1, eu-central-1 (y=135, h=100)
us-east-1 (y=135, h=100) → gw-1, gw-2 (y=310, h=100)

spacing.y = 75

**Formula:**

childY = parentY + parentHeight/2 + spacing.y + childHeight/2

**Level 1 (INTERNET → regions):**
childY = 0 + 10 + 75 + 50 = 135
          │   │    │    └─ child's half (reach center)
          │   │    └────── spacing gap
          │   └─────────── parent's half (reach bottom)
          └─────────────── parent's center

**Level 2 (us-east-1 → gateways):**
childY = 135 + 50 + 75 + 50 = 310

All siblings at the same level share the same Y because they spread horizontally.


## Calculate Y Position (Horizontal Layout - Vertical Stacking)

First we calculate the subtreeHeight, then we calculate the actual `Y` positions of the nodes stacked vertically.
First we need to figure out where the first child starts.
Unlike horizontal spreading where we center children under the parent, here `startY` is already positioned below the parent (parent's bottom + spacing), so we just add half of the first child's height to reach its center.
Following nodes start where the previous node's subtree ended, plus spacing.
So, if our first child's center is at `310`, the next child starts from `310 + 50` (bottom half of first child) `+ 75` (spacing) `+ 50` (top half of second child). And since we only care about the center of the nodes, we accumulate these offsets.

```
us-east-1: y=135, height=100
Children: [gw-1(height=100), gw-2(height=100)]
spacing.y = 75

startY = 135 + 100/2 + 75 = 260

Step 1: First child's center
  y = 260 + 100/2 = 310  ← gw-1's center (no prior siblings)

Step 2: Second child's center (index=1)
  y = 310 (first child's center)
  y += 100/2 = 360 (move to first child's bottom edge)
  y += 75 = 435 (add spacing gap)
  y += 100/2 = 485 ← gw-2's center (move to second child's center)
```

# Connection Path Waypoints

Direct lines between parent and child nodes would overlap and create visual chaos. We need intermediate waypoints to route connections cleanly.

## Horizontal Layout: Trunk + Branch

Parent has children stacked vertically. All children share a vertical trunk line to the left of the parent.

Parent ─┬─ Child1
        ├─ Child2
        └─ Child3
        ^
      trunk

**3-point path**:
trunkX = parentEdges.left - trunkOffset + trunkAdjust

[
  { x: trunkX, y: parentY },           // Move to trunk
  { x: trunkX, y: childY },            // Down trunk to child's Y
  { x: childEdges.left, y: childY }   // Branch right to child
]

All siblings use the same `trunkX`, creating unified tree appearance.

## Vertical Layout: Z-Shape

Children spread horizontally.

```
  Parent
    │
    └──→
        │
    ←──┘
    │
  Child
```

**4-point path**:
midY = parentEdges.bottom + (childEdges.top - parentEdges.bottom) * 0.5

[
  { x: parentX, y: parentEdges.bottom },  // Start at parent bottom
  { x: parentX, y: midY },                // Go down halfway
  { x: childX, y: midY },                 // Move across to child X
  { x: childX, y: childEdges.top }       // Drop to child top
]

## Key Points

- **Attach to edges**, not centers: `getNodeEdges()` calculates left/right/top/bottom from center + dimensions
- **Direction determines shape**: horizontal → trunk, vertical → Z-shape
- **Array order = draw order**: first point to last point

*/
