import { Tooltip } from "@fluentui/react-components";
import {
  Beaker24Regular,
  BranchFork24Regular,
  HeartPulse24Regular,
  Info24Regular,
  PlugConnected24Regular,
  Rocket24Filled,
  Rocket24Regular,
  Settings24Regular,
  Toolbox24Regular,
} from "@fluentui/react-icons";
import { useLayoutEffect, useRef, useState } from "react";
import { useI18n } from "../../i18n/i18n";
import type { AppPage } from "./navigation";

/**
 * Bottom dock.
 *
 * Upstream put navigation in a 64px icon rail on the left, where every entry
 * was a bare glyph that only explained itself on hover. This fork uses a
 * horizontal dock at the bottom with an icon *and* a written label, so the
 * whole app can be driven without discovering tooltips first.
 */
export function DockNavigation({
  page,
  onPageChange,
}: {
  page: AppPage;
  onPageChange: (page: AppPage) => void;
}) {
  const { locale, t } = useI18n();
  const barRef = useRef<HTMLDivElement>(null);
  const activeRef = useRef<HTMLButtonElement | null>(null);
  const [indicator, setIndicator] = useState<{ left: number; width: number } | null>(null);
  const navigationPage = page === "blocked-domains" ? "settings" : page;

  const mainItems = [
    { id: "home", label: t("nav_home"), icon: <Rocket24Regular />, activeIcon: <Rocket24Filled /> },
    { id: "routing", label: t("nav_routing"), icon: <BranchFork24Regular /> },
    { id: "health", label: t("nav_tools"), icon: <HeartPulse24Regular /> },
    { id: "connections", label: locale === "en" ? "Links" : "连接", icon: <PlugConnected24Regular /> },
    { id: "tools", label: locale === "en" ? "Tools" : "工具", icon: <Toolbox24Regular /> },
    { id: "settings", label: t("nav_settings"), icon: <Settings24Regular /> },
  ];

  useLayoutEffect(() => {
    const bar = barRef.current;
    const active = activeRef.current;
    if (!bar || !active) {
      setIndicator(null);
      return;
    }

    const measure = () => {
      // offsetLeft is already relative to the dock bar, which is the offset
      // parent. Reading layout instead of the animated transform keeps the
      // pill and the buttons travelling to the same place.
      setIndicator({ left: active.offsetLeft, width: active.offsetWidth });
    };

    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(bar);
    window.addEventListener("resize", measure);
    return () => {
      observer.disconnect();
      window.removeEventListener("resize", measure);
    };
  }, [navigationPage, locale]);

  return (
    <nav
      className="dock"
      aria-label={locale === "en" ? "Main navigation" : "主导航"}
      onKeyDown={(event) => {
        if (!["ArrowLeft", "ArrowRight", "Home", "End"].includes(event.key)) return;
        const buttons = Array.from(event.currentTarget.querySelectorAll<HTMLButtonElement>(".dock-item:not(:disabled)"));
        const current = buttons.indexOf(event.target as HTMLButtonElement);
        if (current < 0) return;
        event.preventDefault();
        const next = event.key === "Home" ? 0 : event.key === "End" ? buttons.length - 1
          : (current + (event.key === "ArrowRight" ? 1 : -1) + buttons.length) % buttons.length;
        buttons[next]?.focus();
      }}
    >
      <div className="dock-bar" ref={barRef}>
        <span
          className="dock-indicator"
          data-visible={indicator !== null}
          style={{
            transform: `translate3d(${indicator?.left ?? 0}px, 0, 0)`,
            width: indicator ? `${indicator.width}px` : undefined,
          }}
          aria-hidden="true"
        />
        {mainItems.map((item) => {
          const active = item.id === navigationPage;
          return (
            <button
              key={item.id}
              ref={active ? activeRef : undefined}
              type="button"
              className={`dock-item${active ? " is-active" : ""}`}
              aria-label={item.label}
              aria-current={active ? "page" : undefined}
              onClick={() => onPageChange(item.id as AppPage)}
            >
              <span className="dock-item-icon">{active && item.activeIcon ? item.activeIcon : item.icon}</span>
              <span className="dock-item-label">{item.label}</span>
            </button>
          );
        })}
        <span className="dock-divider" aria-hidden="true" />
        {import.meta.env.DEV && (
          <Tooltip content="Appearance Lab" relationship="label" positioning="above">
            <button
              ref={navigationPage === "appearance" ? activeRef : undefined}
              type="button"
              className={`dock-item is-quiet${page === "appearance" ? " is-active" : ""}`}
              aria-label="Appearance Lab"
              aria-current={page === "appearance" ? "page" : undefined}
              onClick={() => onPageChange("appearance")}
            >
              <span className="dock-item-icon"><Beaker24Regular /></span>
              <span className="dock-item-label">{locale === "en" ? "Lab" : "实验"}</span>
            </button>
          </Tooltip>
        )}
        <button
          ref={navigationPage === "about" ? activeRef : undefined}
          type="button"
          className={`dock-item is-quiet${page === "about" ? " is-active" : ""}`}
          aria-label={t("nav_about")}
          aria-current={page === "about" ? "page" : undefined}
          onClick={() => onPageChange("about")}
        >
          <span className="dock-item-icon"><Info24Regular /></span>
          <span className="dock-item-label">{t("nav_about")}</span>
        </button>
      </div>
    </nav>
  );
}
