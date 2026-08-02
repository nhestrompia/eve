import { Image as ImageIcon, Terminal } from "lucide-react";

export function ArtifactUnavailable({
  kind,
}: {
  kind: "image" | "log";
}): React.JSX.Element {
  const Icon = kind === "image" ? ImageIcon : Terminal;
  return (
    <div className="flex min-h-36 flex-col items-center justify-center gap-2 bg-white px-5 text-center text-slate-400">
      <Icon className="size-7" />
      <p className="text-xs font-medium">
        {kind === "image"
          ? "Image unavailable in this checkout"
          : "Log preview unavailable"}
      </p>
    </div>
  );
}
