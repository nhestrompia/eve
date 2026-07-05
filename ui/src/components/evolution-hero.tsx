import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CalendarDays, Download, Tag, Users } from "lucide-react";
import { toast } from "sonner";
import { api } from "../api";
import { humanDate } from "../format";
import type { DetailResponse, SnapshotResponse } from "../types";
import { StatusBadge } from "./status-badge";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "./ui/alert-dialog";
import { Button } from "./ui/button";

export function EvolutionHero({
  detail,
  snapshot,
}: {
  detail: DetailResponse;
  snapshot?: SnapshotResponse;
}) {
  const queryClient = useQueryClient();
  const checkout = useMutation({
    mutationFn: () => api.checkout(detail.summary.id, detail.repository),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: ["config"] });
      if (result.exitCode === 0) {
        toast.success("Snapshot checked out", {
          description: `${result.repository || "Repository"} is now at ${result.commit.slice(0, 12)}.`,
        });
        return;
      }
      toast.error("Checkout failed", {
        description: (
          result.stderr ||
          result.stdout ||
          "EVE could not checkout this snapshot."
        ).trim(),
      });
    },
    onError: (error) => {
      toast.error("Checkout failed", {
        description:
          error instanceof Error
            ? error.message
            : "EVE could not checkout this snapshot.",
      });
    },
  });
  const author = detailAuthor(detail);

  return (
    <section className="py-7 sm:py-9 lg:py-12">
      <div className="flex max-w-[920px] flex-col gap-5">
        <div className="min-w-0">
          <StatusBadge status={detail.summary.status} />
          <h1 className="mt-5 max-w-[14ch] text-[2rem] font-semibold leading-[1.04] tracking-[-0.01em] text-balance sm:text-[2.65rem]">
            {detail.summary.title || "Untitled Snapshot"}
          </h1>
          <p className="mt-5 max-w-[62ch] text-base leading-7 text-pretty text-foreground/90">
            {detail.summary.outcome ||
              detail.evolution.intent ||
              "No outcome recorded."}
          </p>
        </div>

        <div className="flex flex-col gap-4 border-y py-4 min-[860px]:flex-row min-[860px]:items-center min-[860px]:justify-between">
          <div className="flex min-w-0 flex-1 flex-col gap-3 text-sm text-muted-foreground sm:flex-row sm:flex-wrap sm:items-center sm:gap-x-5 sm:gap-y-2 min-[860px]:pr-5">
            <span className="inline-flex min-w-0 items-center gap-2">
              <CalendarDays className="size-4 shrink-0" />
              <span className="min-w-0 truncate">
                {humanDate(
                  detail.summary.updatedAt || detail.summary.createdAt,
                )}
              </span>
            </span>
            <span className="inline-flex min-w-0 items-center gap-2">
              <Users className="size-4 shrink-0" />
              <span className="min-w-0 truncate">Recorded by {author}</span>
            </span>
            <span className="inline-flex min-w-0 items-center gap-2">
              <Tag className="size-4 shrink-0" />
              <span className="min-w-0 truncate">
                Snapshot {detail.summary.id}
              </span>
            </span>
          </div>
          <div className="order-first min-[860px]:order-none min-[860px]:shrink-0">
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button className="h-12 w-full justify-center gap-3 rounded-lg bg-slate-950 px-5 text-white shadow-[0_8px_18px_-14px_rgba(15,23,42,0.7)] hover:bg-slate-900 sm:w-auto sm:min-w-[220px]">
                  <Download className="size-4" />
                  Checkout snapshot
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>
                    Checkout {detail.summary.id}?
                  </AlertDialogTitle>
                  <AlertDialogDescription>
                    This runs{" "}
                    <code className="font-mono">
                      {snapshot?.checkoutCommand ??
                        `eve checkout ${detail.summary.id}`}
                    </code>
                    . EVE will refuse if the working tree is dirty.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction
                    disabled={checkout.isPending}
                    onClick={() => checkout.mutate()}
                  >
                    {checkout.isPending ? "Checking out..." : "Run checkout"}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
            {/* <Button asChild variant="outline" className="h-12 justify-start gap-3 rounded-lg pl-5">
              <Link to="/snapshots/$id/snapshot" params={{ id: detail.summary.id }}>
                <Box className="size-4" />
                View snapshot
              </Link>
            </Button> */}
          </div>
        </div>
      </div>
    </section>
  );
}

function detailAuthor(detail: DetailResponse) {
  const providers = Array.from(
    new Set(
      [
        ...detail.sessions.map(
          (session) => session.providerName || session.provider,
        ),
        ...detail.summary.sessionProviders,
      ].filter(Boolean),
    ),
  );
  if (providers.length > 0) return providers.join(" & ");
  return (
    detail.commits.find((commit) => commit.authorName)?.authorName ||
    "Unknown author"
  );
}
