import { useRef } from "react";
import { getSchedulingStrategy, schedulingStrategies } from "./schedulingStrategies";
import { Badge, Spinner, Dropdown, Option, Tooltip } from "@fluentui/react-components";
import {
  Play20Filled,
  Stop20Filled,
  Globe20Regular,
  ShieldTask20Regular,
} from "@fluentui/react-icons";
import type { EngineMode, EnginePhase } from "../../state/useEngineState";
import { GlassSurface } from "../material/GlassSurface";
import { ThroughputDisplay } from "./ThroughputDisplay";
import { useI18n } from "../../i18n/i18n";

type ModeOption = {
  value: EngineMode;
  title: string;
  summary: string;
  facts: string[];
  caution?: string;
  icon: JSX.Element;
};

export function EngineHero({
  phase,
  mode,
  selectedCount,
  download,
  upload,
  connections,
  history,
  transitioning,
  weighted,
  selectedStrategy,
  onStrategyChange,
  socksPort,
  httpPort,
  systemProxyTakeover,
  telemetryUnavailable = false,
  onModeChange,
  onWeightedChange,
  onToggle,
}: {
  phase: EnginePhase;
  mode: EngineMode;
  selectedCount: number;
  download: number;
  upload: number;
  connections: number;
  history: number[];
  transitioning: boolean;
  weighted: boolean;
  selectedStrategy?: string;
  onStrategyChange?: (value: string) => void;
  socksPort: number;
  httpPort: number;
  systemProxyTakeover: boolean;
  telemetryUnavailable?: boolean;
  onModeChange: (mode: EngineMode) => void;
  onWeightedChange: (value: boolean) => void;
  onToggle: () => void;
}) {
  const { locale, t } = useI18n();
  const text = (zh: string, en: string) => (locale === "en" ? en : zh);
  const strategy = getSchedulingStrategy(weighted, selectedStrategy);
  const language = locale === "en" ? "en" : "zh";
  const groupRef = useRef<HTMLDivElement>(null);

  const active = phase === "running" || phase === "degraded" || phase === "starting";
  const locked = transitioning || phase === "running" || phase === "degraded";
  const actionLabel = phase === "starting"
    ? text("正在启动", "Starting")
    : phase === "stopping"
      ? text("正在停止", "Stopping")
      : phase === "running" || phase === "degraded"
        ? text("停止加速", "Stop")
        : text("开始加速", "Start");
  const phaseLabel = phase === "running"
    ? text("运行中", "Active")
    : phase === "degraded"
      ? text("部分降级", "Degraded")
      : phase === "starting"
        ? text("正在启动", "Starting")
        : phase === "stopping"
          ? text("正在停止", "Stopping")
          : phase === "failed"
            ? text("启动失败", "Failed")
            : text("未运行", "Idle");
  const phaseColor = phase === "running" ? "success"
    : phase === "failed" ? "danger"
      : phase === "degraded" || phase === "starting" ? "warning"
        : "informative";

  const options: ModeOption[] = [
    {
      value: "proxy",
      title: text("轻量", "Light"),
      summary: text("只接管浏览器这类会读系统代理的软件", "Only browsers and apps that follow the system proxy"),
      facts: [text("不用装驱动", "No driver needed"), text("随时可以关", "Easy to undo")],
      caution: text("游戏和部分客户端可能漏过去", "Games and some clients may bypass it"),
      icon: <Globe20Regular />,
    },
    {
      value: "tun",
      title: text("全接管", "Full"),
      summary: text("装一块虚拟网卡，所有软件都走它", "Adds a virtual adapter so every app goes through it"),
      facts: [text("游戏、客户端全覆盖", "Covers games and clients"), text("需要管理员权限", "Needs administrator rights")],
      caution: text("可能与其他 VPN 冲突", "May conflict with other VPNs"),
      icon: <ShieldTask20Regular />,
    },
  ];
  const current = options.find((option) => option.value === mode) ?? options[0];

  const onModeKeyDown = (event: React.KeyboardEvent) => {
    if (!["ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown"].includes(event.key)) return;
    event.preventDefault();
    const next: EngineMode = mode === "proxy" ? "tun" : "proxy";
    onModeChange(next);
    groupRef.current?.querySelector<HTMLButtonElement>(`[data-mode="${next}"]`)?.focus();
  };

  // One sentence that answers "what do I do next?", so the primary action never
  // has to be inferred from the surrounding controls.
  const lead = selectedCount === 0
    ? text("还没选网络，在下面勾一张网卡就可以开始", "No network picked yet — tick one below to start")
    : text(`已选好 ${selectedCount} 条网络`, `${selectedCount} network(s) ready`);

  // The sentence under the switch states what the current mode does to the
  // system, including whether the system proxy was touched.
  const modeDetail = mode === "proxy"
    ? systemProxyTakeover
      ? text("已接管系统代理，浏览器会自动走这里", "System proxy is managed, so browsers route here automatically")
      : text("只在本机开放代理端口，没有改动系统代理设置", "Local proxy ports only; your system proxy setting is untouched")
    : current.summary;

  return (
    <GlassSurface className={`engine-hero phase-${phase}`} aria-label={t("home_engine_control")}>
      <div className="engine-copy">
        <div className="engine-heading">
          <span className="engine-title">{text("网络加速", "Network boost")}</span>
          <Badge key={phase} className="engine-state-badge motion-status-swap" appearance="tint" color={phaseColor}>
            <i className="state-dot" />
            {phaseLabel}
          </Badge>
        </div>
        <p className="engine-lead">{lead}</p>

        <div className="engine-primary">
          <button
            type="button"
            className={`engine-action engine-action-xl${active ? " is-stop" : ""}`}
            disabled={transitioning || (!active && selectedCount === 0)}
            onClick={onToggle}
          >
            {transitioning ? <Spinner size="tiny" /> : active ? <Stop20Filled /> : <Play20Filled />}
            <span key={phase} className="engine-action-label motion-inline-swap">{actionLabel}</span>
          </button>

          <div
            ref={groupRef}
            className="mode-segmented"
            role="radiogroup"
            aria-label={text("切换接管方式", "Switch takeover mode")}
            onKeyDown={onModeKeyDown}
          >
            {options.map((option) => {
              const checked = mode === option.value;
              return (
                <button
                  key={option.value}
                  type="button"
                  role="radio"
                  aria-checked={checked}
                  data-mode={option.value}
                  disabled={locked}
                  tabIndex={checked ? 0 : -1}
                  className={`mode-seg${checked ? " is-active" : ""}`}
                  title={option.summary}
                  onClick={() => onModeChange(option.value)}
                >
                  {option.icon}
                  <span>{option.title}</span>
                </button>
              );
            })}
          </div>
        </div>

        {/* Only the mode in use is explained. Two competing descriptions made
            the choice look like required homework. */}
        <p className="engine-mode-note">
          <strong>{current.title}</strong>
          <span className="engine-mode-sep" aria-hidden="true">·</span>
          {modeDetail}
          {current.caution && <>
            <span className="engine-mode-sep" aria-hidden="true">·</span>
            <span className="engine-mode-caution">{current.caution}</span>
          </>}
        </p>

        <div className="engine-secondary">
          <span className="scheduling-label" id="himix-scheduling-label">{text("流量怎么分", "How traffic splits")}</span>
          <Dropdown
            className="scheduling-dropdown"
            aria-labelledby="himix-scheduling-label"
            value={strategy.label[language]}
            selectedOptions={[strategy.id]}
            disabled={transitioning}
            onOptionSelect={(_, data) => {
              const next = schedulingStrategies.find((item) => item.id === data.optionValue);
              if (next) { if (onStrategyChange) onStrategyChange(next.strategy); else onWeightedChange(next.weighted); }
            }}
          >
            {schedulingStrategies.map((item) => (
              <Option key={item.id} value={item.id}>{item.label[language]}</Option>
            ))}
          </Dropdown>
          <Tooltip
            relationship="description"
            content={mode === "proxy"
              ? text(`HTTP 127.0.0.1:${httpPort} · SOCKS5 ${socksPort}`, `HTTP 127.0.0.1:${httpPort} · SOCKS5 ${socksPort}`)
              : text("启动时会检查路由和权限，可能弹出一次管理员授权。", "Startup checks routes and permissions and may ask for administrator approval once.")}
          >
            <p className="scheduling-hint" tabIndex={0}>
              {strategy.description[language]}
              {active && !transitioning && strategy.strategy !== "latency-first" && <> {text("改动只影响新连接。", "Applies to new connections only.")}</>}
            </p>
          </Tooltip>
        </div>
      </div>
      <ThroughputDisplay download={download} upload={upload} connections={connections} history={history} active={active} unavailable={telemetryUnavailable} />
    </GlassSurface>
  );
}
