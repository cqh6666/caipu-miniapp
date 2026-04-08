# 正式微信登录上线 Checklist

这份清单是按当前仓库状态整理的，目标是把项目从“本地 `dev-login` 联调”切到“正式微信登录 + 真机可用”。

当前项目已知信息：

- 小程序 `AppID`：`wxafe7c4144c9c063e`
- 小程序配置位置：`manifest.json -> mp-weixin.appid`
- 前端登录配置位置：`utils/app-config.js`
- 前端登录实现位置：`utils/auth.js`
- 后端微信登录接口：`POST /api/auth/wechat/login`
- 后端微信配置位置：`backend/configs/example.env`

## 1. 微信后台准备

- [ ] 确认当前发布的小程序就是 `wxafe7c4144c9c063e`
- [ ] 确认你有这个小程序后台的管理员或开发者权限
- [ ] 在微信公众平台补齐小程序基础信息：
  - 名称
  - 头像
  - 简介
  - 服务类目
- [ ] 在微信公众平台的“开发管理”里准备好：
  - `AppID`
  - `AppSecret`
- [ ] 在微信公众平台的“服务器域名”里配置正式接口域名
  - `request` 域名：必须配置
  - `uploadFile` 域名：当前项目有图片上传，必须配置
  - `downloadFile` 域名：如果图片展示走同域静态资源，通常也建议配置
- [ ] 如果服务器部署在中国内地，确认域名已完成 ICP 备案

## 2. 后端准备

- [ ] 在服务器环境变量或 `backend/configs/local.env` 中设置：

```env
WECHAT_APP_ID=wxafe7c4144c9c063e
WECHAT_APP_SECRET=你的小程序AppSecret
JWT_SECRET=请替换成正式随机密钥
APP_ENV=production
APP_ADDR=:8080
```

- [ ] 确认 `WECHAT_APP_ID` 与 `manifest.json` 中的 `mp-weixin.appid` 完全一致
- [ ] 确认后端通过 `Nginx` 或其他方式对外提供 `HTTPS`
- [ ] 确认正式 API 域名可被公网访问，例如：
  - `https://api.xxx.com/healthz`
- [ ] 确认上传静态资源也能通过正式域名访问
  - 当前项目使用 `/uploads/*`

建议的反向代理准备：

- [ ] API 反向代理到 Go 服务
- [ ] `/uploads/` 也能通过同一域名访问
- [ ] 证书有效，浏览器访问无报错

## 3. 前端准备

- [ ] 修改 `utils/app-config.js`

从：

```js
const apiBaseURL = 'http://127.0.0.1:8080'
const requestedAuthMode = 'auto'
```

改成类似：

```js
const apiBaseURL = 'https://api.xxx.com'
const requestedAuthMode = 'wechat'
```

- [ ] 不再使用 `localhost` 或 `127.0.0.1`
- [ ] 保持 `authModeSetting` 为 `wechat`，避免线上仍误走 `dev-login`
- [ ] 重新编译微信小程序产物

## 4. 当前项目里已经做好的保护

这些你不用再额外开发，但上线前要知道：

- [x] 前端调用 `wx.login` 获取临时 `code`
- [x] 前端会把当前小程序 `appId` 一并发给后端
- [x] 后端会校验 `appId` 是否和 `WECHAT_APP_ID` 一致
- [x] 用户首次登录后，后端会自动初始化默认空间，名称会按当前昵称自动生成
- [x] 登录成功后会返回：
  - `token`
  - `kitchens`
  - `currentKitchenId`

## 5. 真机验证清单

- [ ] 真机打开小程序，确认不再走 `dev-login`
- [ ] 首次进入时能成功完成微信登录
- [ ] 首次登录后自动创建默认空间
- [ ] 首页能正常加载菜谱列表
- [ ] 空间页能显示当前空间和成员信息
- [ ] 邀请成员后，分享卡片能正常发给好友
- [ ] 被邀请方打开后能进入邀请页并加入空间
- [ ] 图片上传正常
- [ ] 切换网络环境后仍能重新拉取数据

建议至少用两台设备或两个微信号验证：

- [ ] A 用户首次登录成功
- [ ] B 用户首次登录成功
- [ ] A 邀请 B 加入共享空间
- [ ] B 接受邀请后看到共享空间数据
- [ ] A/B 任一方新增菜谱，另一方刷新后可见

## 6. 隐私与提审准备

- [ ] 在微信后台补齐隐私保护指引
- [ ] 明确声明相册/相机用途
  - 当前项目用于上传菜品成品图
- [ ] 确认提审文案只描述当前已完成能力
  - 微信登录
  - 菜谱记录
  - 图片上传
  - 共享空间邀请

## 7. 发布前最后检查

- [ ] `manifest.json` 中的 `mp-weixin.appid` 正确
- [ ] 前端 `apiBaseURL` 已切正式域名
- [ ] 前端 `requestedAuthMode` 已切 `wechat`
- [ ] 后端 `WECHAT_APP_SECRET` 已配置
- [ ] 微信后台合法域名已配置
- [ ] 真机登录通过
- [ ] 真机上传图片通过
- [ ] 真机邀请加入通过
- [ ] 体验版先上传验证，再提交审核

## 8. 官方文档入口

- `wx.login`
  - <https://developers.weixin.qq.com/miniprogram/dev/api/open-api/login/wx.login.html>
- `code2Session`
  - <https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/user-login/code2Session.html>
- `wx.getAccountInfoSync`
  - <https://developers.weixin.qq.com/miniprogram/dev/api/base/account/wx.getAccountInfoSync.html>
- 小程序网络与服务器域名
  - <https://developers.weixin.qq.com/miniprogram/dev/framework/ability/network.html>
- 小程序隐私保护指引
  - <https://developers.weixin.qq.com/miniprogram/dev/framework/user-privacy/PrivacyAuthorize.html>
- 开发者工具文档入口
  - <https://developers.weixin.qq.com/miniprogram/dev/devtools/devtools.html>
- 开发者工具 CI 文档
  - <https://developers.weixin.qq.com/miniprogram/dev/devtools/ci.html>

## 9. 对你当前项目的最小执行路径

如果你只想先把正式微信登录跑通，按这个最小顺序做：

1. 配好后端 `WECHAT_APP_SECRET`
2. 把前端 `apiBaseURL` 改成正式 `HTTPS` 域名
3. 把 `requestedAuthMode` 改成 `wechat`
4. 在微信后台配置合法域名
5. 真机打开小程序验证首次登录

做到这一步，就已经从“本地开发登录”切到了“正式微信登录”。
