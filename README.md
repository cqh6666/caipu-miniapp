# caipu-miniapp

一个基于 `uni-app` 的微信小程序项目，当前仓库处于初始化阶段，默认页面仍是模板页。

## 当前状态

- 技术栈：`uni-app` + `Vue 3`
- 目标平台：微信小程序
- 当前 `AppID`：`wxafe7c4144c9c063e`
- 当前首页：`pages/index/index.vue`

## 本地开发

### 方式一：使用 HBuilderX

1. 用 HBuilderX 打开项目目录
2. 运行到 `微信小程序`
3. 首次运行时，确保微信开发者工具已登录，并在 `设置 -> 安全设置` 中开启 `CLI/HTTP 调用功能`

### 方式二：使用微信开发者工具

1. 先通过 HBuilderX 编译，或使用项目生成后的微信小程序目录
2. 打开目录：`unpackage/dist/dev/mp-weixin`
3. 在微信开发者工具中导入并运行

## 项目结构

```text
.
├── App.vue
├── main.js
├── manifest.json
├── pages.json
├── pages/
│   └── index/
│       └── index.vue
├── static/
└── uni.scss
```

## 配置说明

- 微信小程序配置位于 `manifest.json` 的 `mp-weixin` 节点
- 编译后的微信项目配置位于 `unpackage/dist/dev/mp-weixin/project.config.json`
- `unpackage/` 是构建产物目录，默认不纳入 Git 管理

## 后续建议

- 替换默认首页模板内容
- 补充接口、页面、组件和业务说明
- 增加发布流程和环境配置说明
