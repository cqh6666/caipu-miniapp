# 微信小程序自动预览脚本

本文说明如何在当前项目里，通过命令行直接触发
“HBuilderX 编译 + 微信开发者工具自动预览”。

## 适用场景

- 日常开发时频繁把最新代码发到手机上预览
- 不想每次都手动执行
  `HBuilderX 编译 -> 微信开发者工具打开项目 -> 点击自动预览`
- 需要在不同 Mac 电脑上复用同一套项目脚本

## 已提供的脚本

项目根目录新增脚本：

- `scripts/wx-auto-preview.sh`

同时在 `package.json` 中补充了两个快捷命令：

- `npm run wx:auto-preview`
- `npm run wx:auto-preview:skip-compile`

## 前置条件

运行脚本前，请先确认：

1. 当前系统是 macOS
2. 已安装 HBuilderX
3. 已安装微信开发者工具
4. 微信开发者工具已登录
5. 微信开发者工具已开启
   `设置 -> 安全设置 -> CLI/HTTP 调用功能`

如果工具没有安装在默认位置，也可以通过环境变量显式指定：

```bash
export HBUILDERX_CLI="/自定义路径/HBuilderX.app/Contents/MacOS/cli"
export WECHAT_DEVTOOLS_CLI="/自定义路径/wechatwebdevtools.app/Contents/MacOS/cli"
```

## 用法

### 1. 正常模式：先编译，再自动预览

```bash
bash scripts/wx-auto-preview.sh
```

或：

```bash
npm run wx:auto-preview
```

脚本会自动执行以下流程：

1. 调用 HBuilderX CLI 编译当前项目到
   `unpackage/dist/dev/mp-weixin`
2. 调用微信开发者工具 CLI 打开该产物目录
3. 调用微信开发者工具 CLI 触发 `auto-preview`

### 2. 跳过编译，直接对现有产物触发自动预览

```bash
bash scripts/wx-auto-preview.sh --skip-compile
```

或：

```bash
npm run wx:auto-preview:skip-compile
```

适合你刚在 HBuilderX 里编译过，只想再次触发手机端自动预览的场景。

### 3. 打开 debug 输出

```bash
bash scripts/wx-auto-preview.sh --debug
```

当自动预览失败时，这个参数更方便排查 CLI 返回信息。

### 4. 显式指定微信开发者工具端口

```bash
bash scripts/wx-auto-preview.sh --port 28964
```

如果你的微信开发者工具已经用固定 HTTP 端口启动，建议命令里显式带上
同一个端口，避免 CLI 连接到别的 IDE 实例。

## 参数说明

| 参数 | 说明 |
| --- | --- |
| `--skip-compile` | 跳过 HBuilderX 编译步骤 |
| `--debug` | 打开微信开发者工具 CLI 的 debug 输出 |
| `--port <port>` | 指定微信开发者工具 HTTP 服务端口 |
| `-h`, `--help` | 查看脚本帮助 |

## 环境变量

| 环境变量 | 说明 |
| --- | --- |
| `HBUILDERX_CLI` | 自定义 HBuilderX CLI 路径 |
| `WECHAT_DEVTOOLS_CLI` | 自定义微信开发者工具 CLI 路径 |
| `WX_CLI_LANG` | 微信 CLI 语言，默认 `zh` |
| `WX_CLI_PORT` | 微信 CLI 端口，可替代 `--port` |

## 脚本定位逻辑

为了尽量兼容不同 Mac 电脑，脚本不会写死当前机器的项目绝对路径，
而是按下面的顺序查找工具：

- 优先读取环境变量里显式指定的 CLI 路径
- 再查默认安装目录：
  - `/Applications/HBuilderX.app`
  - `~/Applications/HBuilderX.app`
  - `/Applications/wechatwebdevtools.app`
  - `/Applications/微信开发者工具.app`
  - `~/Applications/wechatwebdevtools.app`
  - `~/Applications/微信开发者工具.app`
- 若仍未找到，再尝试通过 macOS `mdfind` 查找应用

项目目录则始终以脚本所在仓库自动推导，因此不同开发机只要项目结构一致，
就不需要改脚本内容。

## 常见问题

### 1. 提示未找到某个 CLI

说明脚本没能在默认位置找到工具。优先检查：

- HBuilderX 是否已安装
- 微信开发者工具是否已安装
- 是否安装在非默认路径

若是自定义安装位置，请使用环境变量覆盖。

### 2. `auto-preview` 失败，但 `preview` 能成功

这种情况通常不是项目路径错误，而是微信开发者工具 CLI 当前状态、
HTTP 端口或 IDE 会话状态不一致。可以按下面顺序排查：

1. 确认微信开发者工具已登录
2. 确认已开启 `CLI/HTTP 调用功能`
3. 重新打开一次微信开发者工具项目
4. 显式指定 `--port`
5. 带 `--debug` 再跑一次

### 3. 跳过编译后提示找不到 `project.config.json`

说明当前项目还没有生成微信小程序产物。先执行一次：

```bash
bash scripts/wx-auto-preview.sh
```

或直接在 HBuilderX 里先编译一轮，再使用 `--skip-compile`。

## 相关官方文档

- [微信开发者工具 CLI 文档](https://developers.weixin.qq.com/miniprogram/dev/devtools/cli.html)
- [HBuilderX `launch mp-weixin` 文档](https://hx.dcloud.net.cn/cli/launch-miniProgram?id=launch-mp-weixin)
