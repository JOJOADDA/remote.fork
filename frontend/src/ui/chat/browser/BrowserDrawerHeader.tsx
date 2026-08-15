import type { AgentBrowserStatus, ContainerApp } from "../../../models/project";
import { Crosshair, ExternalLink, Key, Loader, Monitor, RotateCcw, Square, X } from "../../primitives/icons";

type ViewportMode = "fit" | "desktop" | "tablet" | "mobile";

const guiStatusLabel: Record<AgentBrowserStatus, string> = {
  idle: "off",
  starting: "starting…",
  ready: "connected",
  "core-ready": "core ready",
  error: "failed",
  stopped: "stopped",
};

export function BrowserDrawerHeader({
  projectName,
  apps,
  appsLoading,
  selectedPort,
  url,
  canLoad,
  inspectMode,
  guiMode,
  guiStatus,
  fullscreen,
  viewportMode,
  onSelectPort,
  onToggleInspectMode,
  onToggleGuiMode,
  onShowPreview,
  onStopGui,
  onToggleFullscreen,
  onViewportMode,
  onRefresh,
  onClose,
}: {
  projectName: string;
  apps: ContainerApp[];
  appsLoading: boolean;
  selectedPort: number | null;
  url: string;
  canLoad: boolean;
  inspectMode: boolean;
  guiMode: boolean;
  guiStatus: AgentBrowserStatus;
  fullscreen: boolean;
  viewportMode: ViewportMode;
  onSelectPort: (port: number | null) => void;
  onToggleInspectMode: () => void;
  onToggleGuiMode: () => void;
  onShowPreview: () => void;
  onStopGui: () => void;
  onToggleFullscreen: () => void;
  onViewportMode: (mode: ViewportMode) => void;
  onRefresh: () => void;
  onClose: () => void;
}) {
  const previewLabel = selectedPort ? `Live Preview · :${selectedPort}` : "Live Preview";
  const previewState = appsLoading ? "Scanning for apps…" : canLoad ? "Ready to view" : "No running app detected";

  return (
    <header class="codex-header flex-none border-b border-white/10 bg-[#191a1f] px-2.5 py-2 md:px-4 md:py-3">
      <div class="flex min-w-0 items-start gap-2">
        <div class="grid h-10 w-10 flex-none place-items-center rounded-lg border border-white/10 bg-white/[0.06]">
          <Monitor class="h-4 w-4 text-accent-blue" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="flex min-w-0 items-center gap-2">
            <h2 class="truncate text-[15px] font-semibold text-ink-50 md:text-base">
              {guiMode ? "Agent Browser" : previewLabel}
            </h2>
            <span
              class={`h-2 w-2 flex-none rounded-full ${
                (guiMode ? guiStatus === "ready" : canLoad) ? "bg-accent-green" : appsLoading ? "bg-accent-blue animate-pulse" : "bg-ink-400"
              }`}
              aria-hidden="true"
            />
          </div>
          {guiMode ? (
            <div class="truncate text-[12px] text-ink-300" title={`Chrome session · ${guiStatusLabel[guiStatus]}`}>
              {`Chrome session · ${guiStatusLabel[guiStatus]}`}
            </div>
          ) : apps.length > 0 ? (
            <label class="mt-0.5 flex min-w-0 max-w-full items-center gap-1.5 text-[12px] text-ink-300">
              <span class="sr-only">Select running app</span>
              <select
                value={selectedPort ?? ""}
                onChange={(event) => {
                  const value = (event.target as HTMLSelectElement).value;
                  onSelectPort(value ? Number(value) : null);
                }}
                class="min-w-0 max-w-full cursor-pointer rounded bg-transparent py-1 text-[12px] text-ink-300 outline-none hover:text-ink-100 focus:text-ink-100"
                title={url || "Pick a running app"}
              >
                {apps.map((app) => (
                  <option key={app.port} value={app.port} class="bg-[#191a1f] text-ink-100">
                    {appLabel(app)}
                  </option>
                ))}
              </select>
            </label>
          ) : (
            <div class="truncate text-[12px] text-ink-300" title={projectName || undefined}>
              {appsLoading ? "Looking for running apps…" : projectName ? `No apps listening in ${projectName}` : "No project container"}
            </div>
          )}
          {!guiMode && <div class="mt-0.5 truncate text-[10px] text-ink-500" title={url || undefined}>{previewState}{url ? ` · ${url}` : ""}</div>}
        </div>
      </div>

      <div class="mt-2 flex min-w-0 items-center justify-between gap-2 border-t border-white/[0.07] pt-2 md:mt-2">
        <div class="flex min-w-0 shrink items-center gap-1 rounded-lg border border-white/10 bg-white/[0.03] p-0.5">
          <button type="button" onClick={onShowPreview} aria-pressed={!guiMode} class={`rounded-md px-2 py-1.5 text-[11px] font-medium ${!guiMode ? "bg-accent-blue/[0.18] text-accent-blue" : "text-ink-400 hover:text-ink-100"}`}>
            Preview
          </button>
          <button type="button" onClick={onToggleGuiMode} aria-pressed={guiMode} class={`rounded-md px-2 py-1.5 text-[11px] font-medium ${guiMode ? "bg-accent-blue/[0.18] text-accent-blue" : "text-ink-400 hover:text-ink-100"}`}>
            Agent Browser
          </button>
        </div>

        {!guiMode && (
          <div class="flex min-w-0 items-center gap-1 overflow-x-auto">
            {(["fit", "desktop", "tablet", "mobile"] as ViewportMode[]).map((mode) => (
              <button
                key={mode}
                type="button"
                onClick={() => onViewportMode(mode)}
                aria-pressed={viewportMode === mode}
                class={`shrink-0 rounded-md px-2 py-1.5 text-[10px] capitalize ${viewportMode === mode ? "bg-white/[0.12] text-ink-50" : "text-ink-500 hover:text-ink-100"}`}
              >
                {mode}
              </button>
            ))}
          </div>
        )}
      </div>

      <div class="mt-2 flex items-center justify-end gap-1.5 overflow-x-auto pb-0.5">
        {guiMode && (
          <button type="button" onClick={onStopGui} disabled={guiStatus !== "ready"} class="grid h-10 w-10 flex-none place-items-center rounded-lg border border-white/10 bg-white/5 text-ink-200 hover:bg-white/[0.09] disabled:cursor-not-allowed disabled:opacity-50" title="Stop the agent browser" aria-label="Stop the agent browser">
            <Square class="h-4 w-4" />
          </button>
        )}
        {!guiMode && (
          <button type="button" onClick={onToggleInspectMode} disabled={!canLoad} class={`grid h-10 w-10 flex-none place-items-center rounded-lg border disabled:cursor-not-allowed disabled:opacity-50 ${inspectMode ? "border-accent-blue/35 bg-accent-blue/[0.18] text-accent-blue" : "border-white/10 bg-white/5 text-ink-200 hover:bg-white/[0.09]"}`} title="Inspect element" aria-label="Inspect element" aria-pressed={inspectMode}>
            <Crosshair class="h-4 w-4" />
          </button>
        )}
        <button type="button" onClick={onRefresh} disabled={guiMode ? guiStatus !== "ready" : appsLoading} class="grid h-10 w-10 flex-none place-items-center rounded-lg border border-white/10 bg-white/5 text-ink-200 hover:bg-white/[0.09] disabled:cursor-wait disabled:opacity-50" title={guiMode ? "Reload the agent browser" : "Refresh apps and reload preview"} aria-label={guiMode ? "Reload the agent browser" : "Refresh apps and reload preview"}>
          {appsLoading && !guiMode ? <Loader class="h-4 w-4 animate-spin" /> : <RotateCcw class="h-4 w-4" />}
        </button>
        {!guiMode && (
          <a href={canLoad ? url : undefined} target="_blank" rel="noopener noreferrer" aria-disabled={!canLoad} class={`grid h-10 w-10 flex-none place-items-center rounded-lg border border-white/10 bg-white/5 text-ink-200 ${canLoad ? "hover:bg-white/[0.09]" : "pointer-events-none cursor-not-allowed opacity-50"}`} title="Open in new tab" aria-label="Open browser in new tab">
            <ExternalLink class="h-4 w-4" />
          </a>
        )}
        {!guiMode && (
          <button type="button" onClick={onToggleFullscreen} class={`grid h-10 w-10 flex-none place-items-center rounded-lg border border-white/10 bg-white/5 text-ink-200 ${fullscreen ? "bg-accent-blue/[0.18] text-accent-blue" : "hover:bg-white/[0.09]"}`} title={fullscreen ? "Exit fullscreen preview" : "Fullscreen preview"} aria-label={fullscreen ? "Exit fullscreen preview" : "Fullscreen preview"}>
            <Monitor class="h-4 w-4" />
          </button>
        )}
        <button type="button" onClick={onClose} class="grid h-10 w-10 flex-none place-items-center rounded-lg border border-white/10 bg-white/5 text-ink-200 hover:bg-white/[0.09]" title="Close browser" aria-label="Close browser">
          <X class="h-4 w-4" />
        </button>
      </div>
    </header>
  );
}

function appLabel(app: ContainerApp): string {
  const name = app.process?.trim() || "web app";
  return `${name} · :${app.port}`;
}
