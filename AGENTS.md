# Blog 开发约束

## 仓库定位

- `backend/` 是内部博客 API 与浏览器 BFF，使用 Go、SQLite 和 S3 兼容对象存储。
- `frontend/` 是独立博客管理端，使用 Vue 3、TypeScript、Vite 和 Element Plus。
- 浏览器不得持有 Gateway AK/SK；业务请求先到 BFF，再由 BFF 使用 `Gateway-HMAC-SHA256` 调用 Gateway Inner。
- 独立前端通过 People OAuth 登录；统一 `admin-ui` 使用已有 Permission 登录令牌。

## 开发要求

- 后端业务路由只接受 Gateway 重签名请求，业务逻辑放在 `internal/blog`，HTTP 适配放在 `internal/api`。
- 图片等小文件存入 Garage，不得把二进制内容写入 SQLite。
- 后端修改后运行 `gofmt` 和 `go test ./...`；前端修改后运行 `npm run typecheck`、`npm test` 和 `npm run build`。
- 本目录是独立 Git 仓库。验证后提交当前分支；若未配置远端，必须明确报告无法推送。
