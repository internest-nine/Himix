import { Button, Image, Spinner } from "@fluentui/react-components";
import { ArrowSync20Regular, ArrowDownload20Regular, CheckmarkCircle20Filled } from "@fluentui/react-icons";
import { useCallback, useState } from "react";
import { GlassSurface } from "../components/material/GlassSurface";
import { PageHeader } from "../components/shell/PageHeader";
import { productInfo } from "../product";
import { useI18n } from "../i18n/i18n";
import { appServices, type UpdateCheckResult, type UpdateProgress } from "../platform/services";

type CheckState = "idle" | "checking" | "done" | "failed";
type DownloadState = "idle" | "downloading" | "ready";

export function AboutPage() {
  const { locale, t } = useI18n();
  const text = (zh: string, en: string) => (locale === "en" ? en : zh);

  const [checkState, setCheckState] = useState<CheckState>("idle");
  const [result, setResult] = useState<UpdateCheckResult | null>(null);
  const [error, setError] = useState("");
  const [downloadState, setDownloadState] = useState<DownloadState>("idle");
  const [installerPath, setInstallerPath] = useState("");
  const [progress, setProgress] = useState<UpdateProgress | null>(null);

  // Only ever called from the button below. Nothing schedules this, so the app
  // makes no network request until the user explicitly asks for one.
  const checkForUpdates = useCallback(async () => {
    setCheckState("checking");
    setError("");
    setResult(null);
    setDownloadState("idle");
    setInstallerPath("");
    setProgress(null);
    try {
      setResult(await appServices.updater.check());
      setCheckState("done");
    } catch (failure) {
      setError(String(failure));
      setCheckState("failed");
    }
  }, []);

  const downloadUpdate = useCallback(async () => {
    if (!result) return;
    setDownloadState("downloading");
    setError("");
    const timer = window.setInterval(() => {
      void appServices.updater
        .progress()
        .then(setProgress)
        .catch(() => undefined);
    }, 500);
    try {
      setInstallerPath(await appServices.updater.download(result.release));
      setDownloadState("ready");
    } catch (failure) {
      setError(String(failure));
      setDownloadState("idle");
    } finally {
      window.clearInterval(timer);
    }
  }, [result]);

  const installUpdate = useCallback(async () => {
    if (!installerPath) return;
    setError("");
    try {
      await appServices.updater.installAndQuit(installerPath);
    } catch (failure) {
      setError(String(failure));
    }
  }, [installerPath]);

  const percent = progress && progress.total > 0
    ? Math.min(100, Math.round((progress.downloaded / progress.total) * 100))
    : 0;

  return (
    <main className="page about-page">
      <PageHeader
        title={text("关于 Himix", "About Himix")}
        description={t("about_intro")}
      />

      <div className="about-layout">
        <GlassSurface className="about-product">
          <div className="about-brand">
            <Image src="/support/icon.ico" alt={text("Himix 图标", "Himix icon")} className="about-app-icon" />
            <div>
              <h2>{productInfo.name}</h2>
              <strong>{text("当前版本", "Current version")}: v{productInfo.version}</strong>
              <span>{productInfo.build}</span>
            </div>
          </div>
          <p>{t("about_intro")}</p>
        </GlassSurface>

        <GlassSurface className="about-update">
          <h2>{text("更新", "Updates")}</h2>
          <p>
            {text(
              "Himix 不会在启动时或后台自动检查更新。只有你点击下面的按钮时，程序才会访问 GitHub 查询一次有没有新版本。",
              "Himix never checks for updates on startup or in the background. Only when you press the button below does it contact GitHub once to ask whether a newer version exists.",
            )}
          </p>

          <div className="about-update-actions">
            <Button
              appearance="primary"
              icon={<ArrowSync20Regular />}
              onClick={() => void checkForUpdates()}
              disabled={checkState === "checking" || downloadState === "downloading"}
            >
              {checkState === "checking"
                ? text("正在检查…", "Checking…")
                : text("检查更新", "Check for updates")}
            </Button>
            {checkState === "checking" && <Spinner size="tiny" />}
          </div>

          {checkState === "done" && result && !result.available && (
            <p className="about-update-result is-current">
              <CheckmarkCircle20Filled />
              {text(
                `已是最新版本（v${result.current_version}）。`,
                `You are on the latest version (v${result.current_version}).`,
              )}
            </p>
          )}

          {checkState === "done" && result?.available && (
            <div className="about-update-result">
              <p>
                <strong>
                  {text(
                    `发现新版本 v${result.release.tag_name.replace(/^v/i, "")}`,
                    `New version available: v${result.release.tag_name.replace(/^v/i, "")}`,
                  )}
                </strong>
              </p>
              {result.release.notes && (
                <pre className="about-update-notes">{result.release.notes}</pre>
              )}
              <p className="about-update-meta">
                {text("安装包", "Installer")}: {result.release.installer_name}
                {result.release.installer_size > 0 && (
                  <> · {(result.release.installer_size / 1024 / 1024).toFixed(1)} MB</>
                )}
              </p>
              {result.release.installer_digest && (
                <p className="about-update-meta about-update-digest">
                  SHA-256: {result.release.installer_digest.replace(/^sha256:/i, "")}
                </p>
              )}

              {downloadState !== "ready" ? (
                <div className="about-update-actions">
                  <Button
                    appearance="primary"
                    icon={<ArrowDownload20Regular />}
                    onClick={() => void downloadUpdate()}
                    disabled={downloadState === "downloading"}
                  >
                    {downloadState === "downloading"
                      ? text(`正在下载… ${percent}%`, `Downloading… ${percent}%`)
                      : text("下载安装包", "Download installer")}
                  </Button>
                </div>
              ) : (
                <div className="about-update-actions">
                  <Button appearance="primary" onClick={() => void installUpdate()}>
                    {text("安装并退出", "Install and quit")}
                  </Button>
                  <span className="about-update-meta">
                    {text("安装器会关闭 Himix 并接管更新。", "The installer will close Himix and take over.")}
                  </span>
                </div>
              )}
            </div>
          )}

          {checkState === "failed" && (
            <p className="about-update-result is-error" role="alert">
              {text("检查更新失败：", "Update check failed: ")}
              {error}
            </p>
          )}

          {downloadState === "idle" && error && checkState === "done" && (
            <p className="about-update-result is-error" role="alert">
              {text("下载失败：", "Download failed: ")}
              {error}
            </p>
          )}
        </GlassSurface>

        <GlassSurface className="about-copy">
          <h2>{text("关于本构建", "About this build")}</h2>
          <p>
            {text(
              `本程序为 ${productInfo.upstream} 的二次开发版本，遵循 ${productInfo.license} 许可证发布。除你主动点击「检查更新」与下载安装包外，全部功能在本机离线运行：不使用遥测，也不向任何远程服务发送数据。`,
              `This build is derived from ${productInfo.upstream} and released under the ${productInfo.license} license. Apart from the update check and installer download you trigger yourself, everything runs locally and offline: no telemetry, and no data sent to any remote service.`,
            )}
          </p>
          <p>
            {text(
              "原项目版权归其贡献者所有。根据 AGPL-3.0，任何分发本程序的个人或组织须一并提供完整源代码。",
              "Copyright of the upstream project belongs to its contributors. Under AGPL-3.0, anyone distributing this program must also make the complete source available.",
            )}
          </p>
        </GlassSurface>

        <GlassSurface className="sponsor-section">
          <h2>{text("关于支持", "On supporting this project")}</h2>
          <p>
            {text(
              "Himix 免费，没有付费版，也不会做功能限制。维护它花的主要是时间——跟进 Windows 与 Wintun/WFP 的行为变化、在真机上验证安装和升级路径、处理反馈。",
              "Himix is free: no paid tier, no feature locks. What it costs is mostly time — tracking changes in Windows and in Wintun/WFP, verifying install and upgrade paths on real hardware, and working through reports.",
            )}
          </p>
          <p>
            {text(
              "如果它替你省下了一些折腾，而你愿意为这些时间付一点费用，可以扫下面的码。金额随意；不扫也一样，功能上没有任何区别。",
              "If this tool saved you some of that, and you would like to put something toward it, the codes are below. Any amount; or none at all, with nothing changing either way.",
            )}
          </p>
          <div className="payment-grid">
            <figure className="payment-item hm-card">
              <figcaption>{text("微信", "WeChat")}</figcaption>
              <Image src="/support/qr-wechat.png" alt={text("微信赞赏码", "WeChat tip code")} />
            </figure>
            <figure className="payment-item hm-card">
              <figcaption>{text("支付宝", "Alipay")}</figcaption>
              <Image src="/support/qr-alipay.png" alt={text("支付宝赞赏码", "Alipay tip code")} />
            </figure>
          </div>
          <p>
            {text(
              "需要说清楚的是：引擎和绝大部分功能来自上游 HypoMux（作者 Hypostasis-Cat），这个仓库只做产品层的改造与打磨。",
              "Worth saying plainly: the engine and most of the features come from upstream HypoMux, by Hypostasis-Cat. This repository only reworks and polishes the product layer.",
            )}
          </p>
        </GlassSurface>
      </div>
    </main>
  );
}
