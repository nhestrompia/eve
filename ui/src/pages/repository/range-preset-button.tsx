export function RangePresetButton({
  children,
  onClick,
}: {
  children: React.ReactNode;
  onClick: () => void;
}): React.JSX.Element {
  return (
    <button
      type="button"
      onClick={onClick}
      className="rounded-md bg-card px-3 py-2 text-xs font-semibold text-foreground shadow-[var(--shadow-border)] transition-[scale,background-color,box-shadow] hover:bg-accent hover:text-accent-foreground active:scale-[0.97]"
    >
      {children}
    </button>
  );
}
