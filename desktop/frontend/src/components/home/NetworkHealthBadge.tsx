import { Badge } from "@fluentui/react-components";
import { CheckmarkCircle16Filled, Clock16Regular, ErrorCircle16Filled, Warning16Filled } from "@fluentui/react-icons";
import type { AdapterHealth } from "../../state/useEngineState";
import { useI18n } from "../../i18n/i18n";

export function NetworkHealthBadge({ health }: { health: AdapterHealth }) {
  const { locale } = useI18n();
  const labels: Record<AdapterHealth, string> = locale === "en" ? {
    idle: "Not tested",
    healthy: "OK",
    unstable: "Unstable",
    cooldown: "Paused",
    probing: "Probing",
    failed: "Unavailable",
  } : {
    idle: "未测速",
    healthy: "正常",
    unstable: "不稳定",
    cooldown: "暂停",
    probing: "检测中",
    failed: "连不上",
  };
  const icon = health === "failed"
    ? <ErrorCircle16Filled />
    : health === "cooldown" || health === "unstable"
      ? <Warning16Filled />
      : health === "probing" || health === "idle"
        ? <Clock16Regular />
        : <CheckmarkCircle16Filled />;
  const color = health === "healthy" ? "success"
    : health === "failed" ? "danger"
      : health === "unstable" || health === "cooldown" ? "warning"
        : "informative";
  return (
    <Badge
      className="health-badge"
      appearance="tint"
      color={color}
      icon={icon}
    >
      {labels[health]}
    </Badge>
  );
}
