# GrantLedger 公益补助结算规则平台

GrantLedger 是面向公益补助经办团队的年度结算平台。它把费用申报、规则版本、预结算解释、人工复核、确认结算、年度累计、撤销更正、差异分析和受控导出连成可追溯闭环。后端使用 Go 1.24、Gin、pgx、validator、zap、decimal 和 PostgreSQL；中文工作台使用 Vue 3、TypeScript、Vite 与 Pinia。

## 模块职责

- `cmd/server`：组合依赖、启动 HTTP 服务并优雅停机。
- `internal/domain`：固定精度金额、规则版本、申报与复核状态机、分段试算、结算和年度台账。
- `internal/application`：申报、试算、确认、撤销、复核、规则影响、年度重放和导出用例。
- `internal/repository`：完整离线内存事务仓储和 pgx/PostgreSQL 仓储。
- `internal/transport/httpapi`：`/api/v1` 路由、请求校验、分页排序和稳定错误协议。
- `internal/middleware`：request_id、CORS、安全响应头和 panic 恢复。
- `internal/platform`：日志、时钟、ID、本地通知与受控附件存储。
- `internal/service`：离线可验证、幂等可重入的定时事件调度器。
- `migrations`：可重复执行的数据库结构与演示数据。
- `api/openapi`：OpenAPI 3.0 接口描述。
- `web`：管理员工作台、复核队列和试算详情。

## 本地启动

未配置数据库时，服务使用带事务隔离和乐观并发检查的内存仓储，适合离线演示：

```bash
cp .env.example .env
go run ./cmd/server
```

浏览器工作台：

```bash
cd web
npm install
npm run dev
```

服务默认监听 `:8080`，前端默认监听 `:5173` 并代理 `/api`。存活和就绪检查分别是 `/healthz`、`/readyz`。

## PostgreSQL 与迁移

设置 `APP_DATABASE_URL` 后自动切换到 pgx：

```bash
export APP_DATABASE_URL='postgres://postgres:postgres@localhost:5432/grantledger?sslmode=disable'
psql "$APP_DATABASE_URL" -f migrations/001_init.sql
psql "$APP_DATABASE_URL" -f migrations/002_seed.sql
psql "$APP_DATABASE_URL" -f migrations/003_ledger_and_corrections.sql
go run ./cmd/server
```

三份迁移均使用 `IF NOT EXISTS` 或 `ON CONFLICT DO NOTHING`，重复执行不会覆盖已有用户数据或已发布规则版本。生产环境应使用受控迁移账号，并在应用账号上收敛 DDL 权限。

## 配置

| 变量 | 默认值 | 作用 |
|---|---|---|
| `APP_ADDR` | `:8080` | HTTP 地址 |
| `APP_DATABASE_URL` | 空 | 空时使用离线内存仓储 |
| `APP_REQUEST_TIMEOUT` | `5s` | 单请求 context 超时 |
| `APP_SHUTDOWN_TIMEOUT` | `10s` | 优雅停机上限 |
| `APP_ATTACHMENT_ROOT` | `./runtime/attachments` | 附件和导出文件根目录 |
| `APP_MAX_UPLOAD_BYTES` | `5242880` | 附件大小上限 |
| `APP_ALLOWED_ORIGINS` | `http://localhost:5173` | CORS 白名单，逗号分隔 |
| `APP_LOG_LEVEL` | `info` | `debug/info/warn/error` |

本地文件适配器拒绝目录穿越、符号链接和非白名单媒体类型，写入使用同目录临时文件、刷盘和原子替换。通知、回调与定时事件均由本地适配器记录，可在测试中直接核验，不调用第三方接口。

## 演示数据和状态规则

演示数据包括 2026 年困难家庭医疗补助项目、增强保障方案申请人和一版已发布分段规则。申报状态为：

```text
draft -> submitted -> under_review -> approved -> settled
                             |-> need_supplement -> submitted
                             |-> rejected -> under_review
```

例外审批只允许管理员执行。确认结算在同一事务内创建结算、更新申报、占用年度额度、追加台账明细和审计事件；任一步失败都整体回滚。撤销结算同样在一个事务中释放额度、重开申报并保留反向台账和审计原因。

历史规则发布后不可覆盖。试算按费用发生日和保障方案选择已发布版本，逐段计算并返回每段基数、比例、补助额、年度占用前后余额及异常提示。所有金额使用 decimal 字符串，禁止浮点运算。

## 接口示例

创建申报：

```bash
curl -X POST http://localhost:8080/api/v1/claims \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: demo-claim-001' \
  -d '{"claimant_id":"claimant-demo","project_id":"project-care-2026","category":"medical","receipt_summary":"门诊费用","receipt_digest":"sha256:demo-receipt","occurred_on":"2026-05-10","amount":"1200.00"}'
```

分页查询使用 `page`、`page_size`、`sort_by`、`sort_order` 和 `status`。排序只接受白名单字段。错误响应始终包含稳定 `code`、可读 `message`、可选 `field_errors` 和 `request_id`。

## 测试与验证

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./...
cd web && npm test && npm run build
```

领域测试覆盖 decimal 精度、规则版本、分段比例、年度封顶和状态权限；应用测试覆盖幂等、串行化事务、冲突回滚、年度重放和导出；HTTP 测试把请求实际交给 Gin router 并断言响应和仓储副作用；文件测试覆盖目录边界、原子写入、类型与大小限制。

## 安全边界

- API 写操作要求幂等键和乐观版本，数据库再用唯一约束兜底。
- request_id 贯穿响应、日志与审计，panic 恢复不会回显堆栈。
- 日志和审计对密码、令牌、身份号和附件正文做脱敏。
- CSV 导出只允许管理员或审计员，列名使用固定白名单并记录审计事件。
- 仓储 I/O 传递请求 context，服务器配置读写与优雅停机超时。
- `.env`、运行期数据、依赖缓存和构建产物被 `.gitignore` 排除。

