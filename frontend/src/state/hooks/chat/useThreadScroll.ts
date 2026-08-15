import { useEffect, useRef, useState } from "preact/hooks";

export function useThreadScroll(resetKey: string, scrollKey: unknown) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const userScrolledRef = useRef(false);
  const [showJump, setShowJump] = useState(false);

  useEffect(() => {
    unlockAutoScroll();
    setShowJump(false);
  }, [resetKey]);

  useEffect(() => {
    if (userScrolledRef.current) return;
    scrollToBottom("auto");
  }, [scrollKey]);

  useEffect(() => {
    const content = contentRef.current;
    if (!content || typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(() => {
      if (!userScrolledRef.current) scrollToBottom("auto");
    });
    observer.observe(content);
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    const viewport = window.visualViewport;
    if (!viewport) return;
    const handleViewportResize = () => {
      if (!userScrolledRef.current) scrollToBottom("auto");
    };
    viewport.addEventListener("resize", handleViewportResize);
    viewport.addEventListener("scroll", handleViewportResize);
    return () => {
      viewport.removeEventListener("resize", handleViewportResize);
      viewport.removeEventListener("scroll", handleViewportResize);
    };
  }, []);

  function onScroll() {
    const element = scrollRef.current;
    if (!element) return;
    const nearBottom = element.scrollHeight - element.scrollTop - element.clientHeight < 96;
    userScrolledRef.current = !nearBottom;
    setShowJump((current) => (current === !nearBottom ? current : !nearBottom));
  }

  function scrollToBottom(behavior: ScrollBehavior) {
    requestAnimationFrame(() => {
      const element = scrollRef.current;
      if (!element) return;
      element.scrollTo({ top: element.scrollHeight, behavior });
    });
  }

  function unlockAutoScroll() {
    userScrolledRef.current = false;
  }

  function jumpToBottom() {
    unlockAutoScroll();
    setShowJump(false);
    scrollToBottom("smooth");
  }

  return {
    scrollRef,
    contentRef,
    bottomRef,
    showJump,
    onScroll,
    jumpToBottom,
    unlockAutoScroll,
  };
}
