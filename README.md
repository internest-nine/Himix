# Himix

<p align="center">
  <img src="support/icon.ico" alt="Himix 图标" width="128" height="128"><br><br>
  <a href="README.md">简体中文</a> | <a href="README_EN.md">English</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Version-0.1-2F6BFF?style=flat-square" alt="Version 0.1">
  <img src="https://img.shields.io/badge/Base-HypoMux%20v2.8.2-0078D4?style=flat-square" alt="Based on HypoMux v2.8.2">
  <img src="https://img.shields.io/badge/Core-Go-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/Desktop-Wails%20v3-CB3837?style=flat-square" alt="Wails v3">
  <img src="https://img.shields.io/badge/UI-React%20%2B%20Fluent%20UI-61DAFB?style=flat-square&logo=react&logoColor=black" alt="React and Fluent UI">
  <img src="https://img.shields.io/badge/Platform-Windows%2010%20%2F%2011-0078D4?style=flat-square&logo=windows" alt="Windows 10 and 11">
</p>

Himix 是一款面向 Windows 的开源多网卡聚合与分流工具，派生自 [HypoMux v2.8.2](https://github.com/Hypostasis-Cat/HypoMux)（AGPL-3.0）。

它把多连接下载任务分配到多张活动网卡，让有线网络、Wi-Fi、手机热点或 USB 网络共享能够同时承担流量。

**与上游的核心差异：默认不联网、界面重做、路由更省心、附带 Windows 安装包。**
详见下方[与 HypoMux 的差异](#与-hypomux-的差异)。

> 聚合的是多个独立连接，而不是把单条 TCP 连接拆成多路。因此，它尤其适合 Steam、IDM、游戏平台更新器、浏览器大文件下载等高并发场景；单连接任务的速度仍受该连接本身限制。

---

## 下载

到 [Releases](https://github.com/internest-nine/Himix/releases) 页面下载，两种形态功能相同：

| 文件 | 说明 |
| --- | --- |
| `Himix_Setup_0.1.exe` | **安装版（推荐）**。装到 `C:\Program Files\Himix`，自动注册核心服务、创建快捷方式，可完整卸载 |
| `Himix_0.1_portable.zip` | **免安装版**。解压即用，不写注册表、不注册服务；「全接管」模式需每次以管理员身份运行 |

> 安装包**未做代码签名**，Windows SmartScreen 可能提示未知发布者。请对照 Release 页给出的 SHA-256 校验。

---

## 目录

- [下载](#下载)
- [与 HypoMux 的差异](#与-hypomux-的差异)
- [新增功能](#新增功能)
- [核心功能](#核心功能)
- [快速开始](#快速开始)
- [工作原理](#工作原理)
- [技术边界](#技术边界)
- [开发与构建](#开发与构建)
- [开源协议](#开源协议)
- [关于支持](#关于支持)

---

## 与 HypoMux 的差异

上游 v2.8.2 提供了扎实的引擎能力，Himix **完整保留了全部引擎功能**（`engine/internal/` 下 12 个模块一个没少），改动集中在**产品层**。

| 维度 | HypoMux v2.8.2 | Himix 0.1 |
| --- | --- | --- |
| **联网行为** | 内置 AI 助手（可配模型 API）、MCP 外部工具、后台轮询更新 | **AI 与 MCP 全部移除**；更新改为手动触发，不启动检查、不轮询 |
| **界面结构** | 左侧图标导航条（仅图标 + 悬浮提示，带竖向指示条） | **底部一行带文字的胶囊导航**，主区整块留给内容 |
| **模式命名** | 「系统代理模式」「虚拟网卡模式」 | **「轻量」「全接管」**，每张卡片直接写明会发生什么、代价是什么 |
| **规则输入** | 需手动选类型，粘贴网址会报格式错误 | **自动识别**网址 / IP / 进程，直接粘贴即可 |
| **出口选择** | 手动指定网卡 | **自动探测**哪张网卡能通，不通就自动改走能通的 |
| **断线处理** | 指向已掉线网卡的规则会一起失效 | **自动切换**到可用线路，恢复后静默切回 |
| **安装方式** | 需自行获取安装包 | **自带 NSIS 安装包**，装完即用、可完整卸载 |
| **安装包签名** | SignPath 签名（需联网校验证书链） | **未签名**，绿色可审计 |
| **更新方式** | 启动/定时自动检查更新 | **只在点「检查更新」时联网**，清单用 Ed25519 签名校验 |

### 各条差异

**默认不联网。** 上游的 AI 助手支持接入任意模型 API，MCP 还能把数据开放给外部 AI 客户端；更新检查会在启动时和定时访问官方 update channel。Himix 把 AI 与 MCP **整块删除**而非加开关——前端 37 个文件 248 KB 组件、后端 15 个文件 3,687 行代码全部删净，AI 供应商域名零残留，`pixi.js` / `pixi-live2d-display` 两个 Live2D 依赖声明也一并清掉（上游留着它们渲染 AI 皮肤，Himix 已无任何引用）。产物里 `pixi`、`cubism4`、`display`、`AIAssistant` 四个 chunk 彻底消失。更新则改成**只在你点按钮时**查询一次。收益是程序空闲时**一个字节都不往外发**。

> 更新检查是**手动**的：`updater.go` 已注册（约 840 行），但没有任何代码在启动时或定时调用它。清单 `latest.json` 与分离签名 `latest.json.sig` 放在 `update-channel` 分支，客户端用内嵌 Ed25519 公钥验签后才接受。

**界面为内容让路。** 上游用一条左侧图标导航条（`CompactNavigation`，只有图标 + 悬浮提示）来切换页面；Himix 改成底部一行带文字的胶囊导航，主区整块留给卡片。标题栏保持与上游一致的带品牌文字的正常样式。

**路由规则更省心。** 上游加规则要先想清楚"这条该按进程还是域名匹配"；Himix 直接粘贴，它自己判断。配好规则后也不用手动选网卡，它会探测哪条线能通。

**自装更省事。** Himix 提供 NSIS 安装包，双击装到 `C:\Program Files\Himix`，自动注册核心服务、创建快捷方式，卸载时清得干干净净。

---

## 新增功能

### 规则输入自动识别

上游的规则编辑器默认停在「进程」标签页，粘贴一个网址会被当成进程名提交，然后收到引擎的格式错误。Himix 改成**由输入内容本身决定规则类型**：

```
https://www.example.com/item/123?id=5   →  域名规则 example.com（涵盖其下所有站点）
192.168.1.0/24                          →  IP/CIDR 规则
D:\Games\steam.exe                      →  进程规则
```

边界情况（裸域名、用户信息、尾点、国际化域名转 punycode、带端口、带路径、CIDR 前缀、IP、中文域名、非法输入、路径转可执行文件名）由 21 项单元测试覆盖。

### 出口自动探测与自动选路

加完规则后，Himix 会**主动探测每张网卡能否到达目标**，并把探测结果显示出来。两种情况会自动替你做决定：

- **当前选的那条线确实不通** → 自动改走能通的那条线
- **处于聚合模式，但只有一条线能通** → 改为固定到那条线

第二条尤其重要：聚合模式里混进一条死链，会让内网站点**时好时坏**，比固定到唯一可用线路更糟。

### 断线自动切换

网卡从机器上掉线时，指向它的规则不应该跟着一起失效。Himix 的做法是：

- 保存的规则**保留**原本选择的网卡
- 该网卡不可用期间，规则临时**投影到聚合池**——而此时池子里只剩还活着的线路
- 线路恢复、或重启程序后，**自动还原**原始路由，用户无需任何操作

这条路径复用了正常的规则发布流程（`writeSingBoxRuleSetPlanLocked`），而不是另开一套写入逻辑，因此行为与常规路径完全一致。

### 界面重做

- **模式选择从技术名词改为用户视角**：上游叫「系统代理模式」「虚拟网卡模式」；Himix 改成「轻量」与「全接管」，每张卡片直接写明会发生什么、代价是什么——「轻量」写着"不用装驱动""随时可以关"，代价是"游戏和部分客户端可能漏过去"；「全接管」写着"游戏、客户端全覆盖""需要管理员权限"，代价是"可能与其他 VPN 冲突"
- **导航用词更短**：上游叫「首页 / 路由规则 / 网络检测 / 系统设置」，Himix 改成「加速 / 分流 / 体检 / 设置」
- **规则输入框直接给例子**：上游只写「例如 cs2.exe」「例如 example.com（包含子域名）」，Himix 改成「程序名或完整路径，例如 cs2.exe」与「粘贴网址就行，例如 https://www.example.com/item/123」
- **品牌主色** `#2F6BFF`，卡片圆角 20px，去掉装饰性分隔线与光晕
- 找不到可用网卡时给出可执行的提示（「请检查网卡是否已连接并具有可用 IPv4 地址，然后重新扫描」）并配一个重新扫描按钮，而不是留空
- 外观系统：Mica / 玻璃 / 纯色材质、6 种强调色、壁纸与背景调节、密度与动效档位

### Windows 安装包

NSIS 打包，4 页中文向导：

- 装到 `C:\Program Files\Himix`，自动注册 `HimixCore` 核心服务
- 自动创建桌面与开始菜单快捷方式
- **安装事务**：先快照文件与注册表，再应用变更；任一步失败自动回滚，不会留下半装状态
- 核心服务装在 `%ProgramData%\Himix\Core`，目录 ACL 只给 LocalSystem 与管理员写权限，普通用户只读
- 卸载彻底：安装目录、注册表项、两个快捷方式全部清除

### 发布链路修复

上游的改名与发布脚本存在多处不一致，本版一并修好：

- **安装包名与更新器对齐**：安装包统一为 `Himix_Setup_<版本>.exe`
- **引擎 CI 指向真实目录**：修好了指向已重命名目录导致的构建失败
- **发布前验证不再被静默跳过**：修正环境变量名不一致导致测试 `t.Skip()` 的问题
- 新增跨侧契约测试，比对 NSIS 脚本 / 引擎 / 桌面三方的目录名、可执行文件名、服务名、管道名与注册表键

### 版本号支持两段写法

版本号定为 `0.1`（两段）。缺失的补丁位在所有比较中按 0 计算（`0.1` 与 `0.1.0` 比较结果一致），但字符串保留原始写法，保证**发布标签、元数据、安装包名三者始终一致**。

---

## 核心功能

以下能力完整继承自上游 v2.8.2，Himix 未做删减：

- **系统代理模式**：启动本地 HTTP/HTTPS 与 SOCKS5 服务；默认接管 Windows 系统代理，也可在设置中关闭接管、仅向下载器等手动配置的应用提供本地端口
- **虚拟网卡模式**：通过 Wintun 与 sing-box 接管更广泛的系统流量，并结合 WFP、DNS 和路由规则完成精细分流
- **多网卡连接调度**：为每个新连接选择出站网卡，并使用源地址绑定与 `IP_UNICAST_IF` 将套接字固定到真实物理链路
- **高级路由规则**：按进程、域名、IP/CIDR 指定聚合、直连、以太网、Wi-Fi 或某张具体网卡
- **单网卡受限域名处理**：记录特定链路不可访问的域名，避免后续连接继续分配到该链路
- **加密 DNS**：支持 DoH 与自定义 DoT，多服务器、IPv4/IPv6、指定 DNS 出口，校验服务器证书且不自动降级明文
- **实时状态与诊断**：展示每张网卡的速率、连接数和聚合吞吐，并提供丢包、延迟、抖动、DNS、网关及源地址绑定检查
- **最小权限架构**：界面不以管理员身份常驻；仅独立 Core 服务持有必要的网络管理权限
- **Steam 下载优选**、**热点面板**、**规则集面板**、**连接查看**、**MTU / NAT 探测**、**拦截域名**等工具

### 两种运行模式如何选择

| 模式 | 覆盖范围 | 权限与兼容性 | 推荐场景 |
| --- | --- | --- | --- |
| 轻量模式 | 遵循 Windows 系统代理的应用，或手动填写本地 HTTP/SOCKS5 端口的应用 | 更轻量；可关闭系统代理接管；不创建虚拟网卡 | IDM、浏览器、Steam 等代理感知型下载 |
| 全接管模式 | 更广泛的 TCP/UDP 与非代理感知流量 | 需要 Core 服务、Wintun/WFP；不能与其他 TUN 同时接管默认路由 | 游戏平台下载、WeGame、复杂分流和全局接管 |

---

## 快速开始

1. 让电脑同时连接至少两条可用网络，例如有线宽带 + Wi-Fi，或宽带 + 手机 USB/热点共享
2. 安装 Himix（双击 `Himix_Setup_0.1.exe`）或直接运行 `himix.exe`
3. 启动后，在首页刷新并勾选要加入聚合池的活动网卡
4. 先运行「网络体检」，确认各链路具有有效 IPv4、网关、DNS 和源地址绑定能力
5. 选择轻量模式或全接管模式。低延迟游戏、语音和会议程序可先加入直连规则
6. 开启聚合引擎，再启动下载任务。若 Steam 已经运行，按界面提示重启 Steam 以完整应用代理设置
7. 使用完毕后停止聚合或正常退出，Himix 会恢复其接管的系统网络设置

---

## 工作原理

```text
应用连接
   │
   ├─ 轻量模式：HTTP/HTTPS 10801 · SOCKS5 10800
   └─ 全接管模式：Wintun + sing-box + WFP/DNS/路由规则
                         │
                  himix-engine.exe
                         │
              按连接选择聚合/直连/指定网卡
               ┌─────────┼─────────┐
             网卡 1     网卡 2     网卡 3
               └─────────┴─────────┘
                     合并吞吐
```

轻量模式将本地代理链写入当前用户的 Windows Internet Settings；全接管模式则通过独立权限服务创建和管理 TUN、WFP、DNS 与路由资源。Go 引擎对每个新连接选择出口，把本地源地址绑定和 Windows 接口索引绑定组合使用，从而让不同连接稳定地走不同物理网卡。

---

## 技术边界

- 适合 IDM、Steam、Epic Games Launcher、EA App、Xbox、WeGame、Chrome、Edge、Firefox 等会产生多个并发连接的下载场景
- 多网卡聚合是**连接级负载分配**，不会让单条 TCP 连接突破其原有链路速度，也不会把多条线路变成一个具有单一公网 IP 的链路聚合协议
- 聚合侧重吞吐量，不保证降低延迟。竞技游戏、语音和视频会议建议使用直连规则或暂时停止聚合
- Himix 不读取游戏内存、不注入 DLL，也不修改游戏私有协议数据包；但第三方平台和反作弊规则各不相同，用户仍应遵守相应服务条款
- 仅可在本人拥有或已获授权的设备与网络上使用，不得用于绕过未经授权的访问控制、安全措施或平台限制
- **本版安装包未做代码签名**，Windows SmartScreen 可能提示未知发布者

### ⚠️ 重要提示与合规免责声明

Himix 是一个透明、开源的网络工具，仅用于用户本人拥有授权的设备与网络连接。它不应用于绕过第三方访问控制、网络限制、平台规则或任何未经授权的安全措施。

使用前请确认你理解以下行为边界：

1. **系统设置调整**：运行时可能动态调整 Windows 系统代理和/或路由相关设置，以便将流量导入加速核心
2. **本地安全代理**：启用加速后，需要加速的网络流量会经过本机安全核心进行分流、代理与多路复用
3. **自动恢复机制**：停止工具或卸载软件时，会自动恢复被修改的系统代理与网络设置
4. **游戏与分流规则**：对竞技类网游等对延迟极度敏感的应用，建议将其加入**直连/绕过规则列表**，或玩游戏时暂停本工具

---

## 开发与构建

```bash
# 前端依赖
cd desktop/frontend && pnpm install

# 开发模式（热重载）
cd desktop && wails3 task dev

# 生产构建 → bin/himix.exe
cd desktop && wails3 task build

# 打包 NSIS 安装包（需先安装 NSIS 并设置 MAKENSIS）
export MAKENSIS="/c/Program Files (x86)/NSIS/makensis.exe"
cd desktop && wails3 task windows:package

# 仅当前用户安装（免管理员）
cd desktop && wails3 task windows:package INSTALL_SCOPE=user
```

**环境要求**：Go 1.26+ · Node 22+ · pnpm · Wails v3 · NSIS 3.13+

**测试**：

```bash
cd desktop && go test ./... -count=1
cd desktop/frontend && npx tsc --noEmit && npx vitest run
cd engine && go test ./... -count=1
```

### 目录结构

```
src/
├── desktop/          Wails 桌面端（Go 后端 + React 前端）
│   ├── frontend/     React + TypeScript + Fluent UI
│   ├── internal/
│   │   ├── services/ 业务服务（路由、连接、设置、诊断…）
│   │   ├── engineclient/ 与核心的通信（含提权握手）
│   │   ├── platform/ 平台适配与托盘
│   │   └── releaseversion/ 版本号解析与元数据同步
│   ├── build/windows/nsis/  安装包脚本
│   └── cmd/          release-version / update-manifest-sign
├── engine/           独立的 Go 网络引擎（TUN、WFP、DNS、代理、调度）
├── protocol/         协议定义
└── .github/workflows/ CI：构建、签名、发布、发布冒烟
```

### 版本号约定

版本号写在 `desktop/VERSION`，用 `go run ./cmd/release-version -write` 同步到全部 12 处元数据
（Taskfile、package.json、product.ts、updater.go、config.yml、version.nsh、info.json、manifest、msix…）。
**不要手改单个文件**，必然漏。

版本号支持 `X.Y` 与 `X.Y.Z` 两种写法。缺省补丁位在比较中按 0 计算，但字符串保留原始写法，
保证发布标签、元数据与安装包名三者一致。

---

## 开源协议

Himix 派生自 [HypoMux](https://github.com/Hypostasis-Cat/HypoMux)，遵循 **AGPL-3.0** 许可证发布（见 [`LICENSE`](LICENSE)）。

本项目基于上游 [v2.8.2](https://github.com/Hypostasis-Cat/HypoMux/tree/v2.8.2) 的源码二次开发（对应提交 `7667cbf9`）。上游完整源码可在该标签处获取，本仓库未附带其副本。

> **AGPL-3.0 的实际约束**：自用、内部使用无任何限制；一旦对外分发（包括发给同事、在团队内部共享二进制），衍生作品必须同样以 AGPL-3.0 开源并提供完整源代码。

上游项目的赞助方、签名服务与致谢名单见上游 README。本分支不重复列出。

---

## 致谢

- **[Hypostasis-Cat](https://github.com/Hypostasis-Cat)** — 上游 HypoMux 的作者与维护者。本项目基于其 v2.8.2 发布，引擎层的绝大部分工作来自上游。
- **[SignPath.io](https://signpath.io/)** — 为上游提供免费 Windows 代码签名服务。本分支的安装包未签名，故不涉及。
- 所有为上游贡献代码、报告问题与提供反馈的人。

<p align="center">
  <a href="https://github.com/Hypostasis-Cat/HypoMux"><img src="https://img.shields.io/badge/upstream-HypoMux%20v2.8.2-0078D4?style=flat-square" alt="Upstream"></a>
</p>

---

## 关于支持

Himix 免费，没有付费版，也不会做功能限制。

维护它花的主要是时间：跟进 Windows 与 Wintun/WFP 的行为变化、在真机上验证安装和升级路径、处理反馈。如果这个工具替你省下了一些折腾，而你愿意为这些时间付一点费用，下面的码就在这儿。金额随意；不扫也一样，功能上没有任何区别。

需要说清楚的是：引擎和绝大部分功能来自上游 HypoMux，这个仓库只做产品层的改造与打磨。

比起赞赏，反馈更有价值——开 [Issue](https://github.com/internest-nine/Himix/issues) 时附上「关于」页里的诊断信息，能省掉大半来回。

<table align="center">
  <tr>
    <th align="center">微信</th>
    <th align="center">支付宝</th>
  </tr>
  <tr>
    <td align="center"><img src="support/qr-wechat.png" width="220" alt="微信赞赏码"></td>
    <td align="center"><img src="support/qr-alipay.png" width="220" alt="支付宝赞赏码"></td>
  </tr>
</table>

