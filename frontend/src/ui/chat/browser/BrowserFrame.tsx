import { useEffect, useState } from "preact/hooks";
import type { RefObject } from "preact";
import { BrowserEmptyState } from "./BrowserEmptyState";

type ViewportMode = "fit" | "desktop" | "tablet" | "mobile";

export function BrowserFrame({
  canLoad,
  iframeRef,
  iframeUrl,
  reloadKey,
  projectName,
  resizing,
  inspectMode,
  onFrameLoad,
  viewportMode,
}: {
  canLoad: boolean;
  iframeRef: RefObject<HTMLIFrameElement>;
  iframeUrl: string;
  reloadKey: number;
  projectName: string;
  resizing: boolean;
  inspectMode: boolean;
  onFrameLoad: (enabled: boolean) => void;
  viewportMode: ViewportMode;
}) {
  const [loading, setLoading] = useState(canLoad);

  useEffect(() => {
    setLoading(canLoad);
  }, [canLoad, iframeUrl, reloadKey]);

  if (!canLoad) {
    return <BrowserEmptyState />;
  }

  const frameWidth =
    viewportMode === "mobile"
      ? "w-[390px] max-w-full"
      : viewportMode === "tablet"
        ? "w-[768px] max-w-full"
        : viewportMode === "desktop"
          ? "w-[1200px] max-w-full"
          : "w-full";

  return (
    <div class="relative flex min-h-0 flex-1 items-start justify-center overflow-auto bg-[#0d1015] p-2 sm:p-4">
      <div class={`${frameWidth} relative min-h-full overflow-hidden rounded-lg bg-white shadow-2xl ring-1 ring-white/10 transition-[width] duration-200`}>
        {loading && (
          <div class="absolute inset-0 z-10 grid place-items-center bg-[#101318] px-6 text-center" role="status" aria-live="polite">
            <div>
              <div class="mx-auto mb-3 h-7 w-7 animate-spin rounded-full border-2 border-white/15 border-t-accent-blue" />
              <p class="text-sm font-medium text-ink-200">Loading live preview</p>
              <p class="mt-1 text-xs text-ink-500">Connecting to the app running in {projectName || "the project container"}.</p>
            </div>
          </div>
        )}
        <iframe
          ref={iframeRef}
          key={`${iframeUrl}:${reloadKey}`}
          src={iframeUrl}
          title={`Live preview for ${projectName || "container"}`}
          onLoad={() => {
            setLoading(false);
            onFrameLoad(inspectMode);
          }}
          onError={() => setLoading(false)}
          class={`h-full min-h-[calc(100vh-13rem)] w-full border-0 bg-white ${resizing ? "pointer-events-none" : ""}`}
          allow="clipboard-read; clipboard-write"
        />
      </div>
    </div>
  );
}
