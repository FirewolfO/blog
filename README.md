# 内部博客

内部 Blog 系统包含 Go 后端和 Vue 3 前端。独立前端通过 People OAuth 登录；统一 Admin UI 复用 Permission 登录令牌，并由 Blog BFF 强制校验 `svc.inner.blog:view` / `svc.inner.blog:manage`。浏览器请求先到 Blog BFF，BFF 使用服务 AK/SK 签名后调用 Gateway Inner，再由 Gateway 转发到 Blog 内部业务接口。

## 能力

- 文章创建、编辑、检索、草稿、发布、归档与删除
- Markdown 正文、摘要、标签、分类和封面图
- 分类维护与占用保护
- 投稿审核、作者审核记录与审核结果通知
- 文章评论与评论删除
- 工作台统计与最近更新
- Garage S3 兼容小文件存储，图片通过 `/media/{id}` 稳定访问

## 目录

- `backend/`：Go API、BFF、SQLite 数据库与 Garage 客户端
- `frontend/`：独立博客管理页面，固定端口 `5179`
- `storage/`：Garage `v2.3.0` 本地安装和启停脚本

## 本地运行

首次启动会从 Garage 官方发布地址下载 Linux AMD64 二进制到 `.runtime/bin/garage`，生成随机 RPC/Admin 密钥，并把元数据和对象持久化到 `.runtime/garage/`。

```bash
npm --prefix frontend install
./start.sh
```

服务地址：

- Blog UI：`http://localhost:5179`
- Blog API：`http://localhost:8086`
- Garage S3：`http://127.0.0.1:3900`
- Garage Web：`http://127.0.0.1:3902`
- Garage Admin：`http://127.0.0.1:3903`

媒体接口返回相对 URL `/media/{id}`，两个前端开发服务器均已配置代理，因此通过 localhost 或工作区 IP 访问时都能正确展示图片。

停止服务：`./stop.sh`。所有生产环境密钥必须通过环境变量覆盖，Garage 单节点模式没有数据冗余，不应作为高可用生产部署。

## 验证

```bash
cd backend && go test ./...
cd ../frontend && npm run typecheck && npm test && npm run build
GARAGE_CONFIG_FILE=../.runtime/garage.toml ../.runtime/bin/garage status
```
