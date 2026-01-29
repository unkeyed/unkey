// Generate a consistent gradient based on a string (email or name)
export function getGradientForUser(identifier: string): { from: string; to: string } {
  const gradients = [
    { from: "#3b82f6", to: "#8b5cf6" }, // bright blue to vivid purple
    { from: "#10b981", to: "#06b6d4" }, // emerald to cyan
    { from: "#ec4899", to: "#f43f5e" }, // hot pink to rose
    { from: "#f59e0b", to: "#ef4444" }, // amber to red
    { from: "#6366f1", to: "#3b82f6" }, // indigo to blue
    { from: "#14b8a6", to: "#22c55e" }, // teal to green
    { from: "#a855f7", to: "#ec4899" }, // purple to pink
    { from: "#f97316", to: "#eab308" }, // orange to yellow
  ];

  // Simple hash function to get consistent index
  let hash = 0;
  for (let i = 0; i < identifier.length; i++) {
    hash = identifier.charCodeAt(i) + ((hash << 5) - hash);
  }
  const index = Math.abs(hash) % gradients.length;
  return gradients[index];
}
