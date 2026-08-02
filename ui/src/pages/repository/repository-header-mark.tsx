import type { RepositorySummary } from "../../types";

export function RepositoryHeaderMark({
  repository,
}: {
  repository: RepositorySummary;
}): React.JSX.Element {
  if (repository.name.toLowerCase() === "eve") {
    return (
      <div className="flex h-[70px] w-[170px] shrink-0 items-center rounded-lg bg-white px-3 ring-1 ring-inset ring-slate-200">
        <img
          src="/eve.svg"
          alt="eve"
          className="eve-logo h-full w-full object-contain object-left"
        />
      </div>
    );
  }

  return (
    <div
      className="grid h-[70px] w-[170px] shrink-0 place-items-center rounded-lg bg-slate-950 text-white ring-1 ring-inset ring-slate-800"
      aria-label={`${repository.name} repository mark`}
    >
      <span className="text-[34px] font-semibold uppercase leading-none tracking-normal">
        {repositoryInitial(repository.name)}
      </span>
    </div>
  );
}

export function repositoryInitial(name: string): string {
  const firstLetter = name.trim().match(/[A-Za-z0-9]/)?.[0];
  return (firstLetter ?? "?").toUpperCase();
}
