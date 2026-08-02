import type { RepositoryTab } from "./types";

export function RepositoryTabNavigation({
  activeTab,
  tabs,
  onActiveTabChange,
}: {
  activeTab: RepositoryTab;
  tabs: Array<{ id: RepositoryTab; label: string; count?: number }>;
  onActiveTabChange: (tab: RepositoryTab) => void;
}): React.JSX.Element {
  return (
    <div
      className="flex gap-7 overflow-x-auto overflow-y-hidden text-sm font-medium text-muted-foreground"
      role="tablist"
      aria-label="Repository sections"
    >
      {tabs.map((tab) => (
        <button
          key={tab.id}
          type="button"
          role="tab"
          aria-selected={activeTab === tab.id}
          aria-controls={`repository-tab-${tab.id}`}
          id={`repository-tab-trigger-${tab.id}`}
          data-state={activeTab === tab.id ? "active" : "inactive"}
          onClick={() => {
            onActiveTabChange(tab.id);
            window.history.replaceState(
              null,
              "",
              tab.id === "overview"
                ? window.location.pathname
                : `#${tab.id}`,
            );
          }}
          className={`relative inline-flex min-h-12 shrink-0 items-center gap-2 rounded-t-lg px-3 text-left transition-colors after:absolute after:inset-x-0 after:bottom-0 after:h-0.5 after:rounded-full hover:bg-slate-50 hover:text-foreground data-[state=active]:bg-white data-[state=active]:after:bg-blue-600 ${
            activeTab === tab.id
              ? "text-blue-700"
              : "text-muted-foreground after:bg-transparent"
          }`}
        >
          <span>{tab.label}</span>
          {tab.count !== undefined ? (
            <span
              className={`rounded-full px-2 py-0.5 text-xs ${
                activeTab === tab.id
                  ? "bg-blue-50 text-blue-700"
                  : "bg-slate-100 text-slate-500"
              }`}
            >
              {tab.count}
            </span>
          ) : null}
        </button>
      ))}
    </div>
  );
}
