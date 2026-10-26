import { styles } from "./theme";

export function Emphasis({ children }: { children: string }) {
  return (
    <strong className="email-text" style={styles.emphasis}>
      {children}
    </strong>
  );
}
