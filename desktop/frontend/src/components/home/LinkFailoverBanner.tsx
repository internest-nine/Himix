import { useI18n } from "../../i18n/i18n";
import type { LinkFailoverState } from "../../platform/services";

/**
 * Shown while a NIC this session was using is gone. The engine keeps the saved
 * rules and the saved adapter selection; this banner reports what it did at
 * runtime, because otherwise the UI would look untouched while traffic quietly
 * moved to another line.
 */
export function LinkFailoverBanner({ state }: { state?: LinkFailoverState }) {
  const { locale } = useI18n();
  const text = (zh: string, en: string) => (locale === "en" ? en : zh);
  if (!state?.degraded) return null;

  const lost = (state.lost_adapters ?? []).filter(Boolean);
  const active = (state.active_adapters ?? []).filter(Boolean);
  const lostLabel = lost.join(locale === "en" ? ", " : "、");
  const activeLabel = active.join(locale === "en" ? ", " : "、");
  const recovering = Boolean(state.recovering);
  const seconds = Math.max(1, Math.ceil((state.recover_in_ms ?? 0) / 1000));

  let title = text(`${lostLabel} 已断开`, `${lostLabel} disconnected`);
  if (!lost.length) title = text("有线路断开了", "A line disconnected");
  if (recovering) title = text(`${lostLabel || "线路"} 已恢复`, `${lostLabel || "Line"} is back`);

  let detail = activeLabel
    ? text(`流量已转到 ${activeLabel}，新连接自动走这条线`, `Traffic moved to ${activeLabel}; new connections use it`)
    : text("当前没有可用线路，网络恢复后会自动接上", "No usable line right now; it will reconnect on its own");
  if (recovering) {
    detail = activeLabel
      ? text(`再稳定 ${seconds} 秒就切回，期间仍有流量走 ${activeLabel}`, `Switching back in ${seconds}s; traffic keeps using ${activeLabel}`)
      : text(`再稳定 ${seconds} 秒就切回`, `Switching back in ${seconds}s`);
  }

  return (
    <div
      className={`link-failover-banner${recovering ? " is-recovering" : ""}${active.length ? "" : " is-offline"}`}
      role="status"
      aria-live="polite"
    >
      <span className="link-failover-dot" aria-hidden="true" />
      <span className="link-failover-text">
        <strong>{title}</strong>
        <span>{detail}</span>
        {state.note ? <span className="link-failover-note">{state.note}</span> : null}
      </span>
    </div>
  );
}
