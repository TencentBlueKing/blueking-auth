# KMS 凭证配置

BKAuth 支持在启动时从本地 KMS 信封解密数据库、Redis、应用密钥和加密密钥等凭证，替代配置文件中的明文。默认关闭，关闭时保持原有的配置读取行为，不读取信封。

## 配置

KMS 的开关、私钥和信封路径都写在配置文件里，不使用环境变量：

```yaml
kms:
  enabled: true
  # Base64 编码的 PEM 私钥内容（不是路径），支持 RSA（PKCS#8 / PKCS#1）与 SM2
  privateKey: "<base64 pem>"
  # 信封文件路径；留空时使用默认值 /etc/secrets/bkauth-kms
  envelopeFile: "/etc/secrets/bkauth-kms"
```

- `kms.enabled` 未配置或为 `false` 时保持原有的配置读取行为，不读取信封。
- `kms.privateKey` 在开启时必填，为空则启动失败。
- 信封文件内容为 KMS 签发的 Base64 信封，首尾空白会被忽略。

Helm Chart 开启 KMS 后把 `privateKey` 渲染进配置，并把信封文件挂载到 `envelopeFile` 指定的路径。

## 生效方式

- 解密发生在 `config.Load()` 中，`bkauth` 服务及所有子命令（`fixture_init`、`cli`、`secret_encryption` 等）共用同一入口。
- 解密结果只覆盖内存中的配置，不写回环境变量或配置文件；地址、端口、库名、TLS 证书路径、`crypto.nonce` 等非凭证配置仍从配置文件读取。
- 开启后，信封中缺少某个必需字段、字段不是字符串、或为空 / 纯空白时，进程启动失败，**不会回退到配置文件中的旧值**。
- 凭证值按原文使用（保留首尾空白和 `$` 等字符，不做变量替换）。
- 错误信息只包含失败阶段或字段路径（如 `mysql.bkauth.password`），不包含凭证内容。

## 信封模板

以下是**加密前的 JSON 明文结构**，不是挂载文件的最终内容。占位值必须替换为当前部署正在使用的凭证。

```json
{
  "bkapp_id_secret": {
    "default": {"app_code": "bk_auth", "app_secret": "<bkauth app secret>"}
  },
  "mysql": {
    "bkauth": {"username": "<database user>", "password": "<database password>"}
  },
  "redis": {
    "default": {"password": "<redis password>", "sentinel_password": "<sentinel password>"}
  },
  "encryption": {
    "cryptoKey": "<existing crypto.key>"
  }
}
```

| 信封路径 | 对应配置 | 必填条件 |
| --- | --- | --- |
| `bkapp_id_secret.default.app_secret` | `appSecret` | 总是 |
| `encryption.cryptoKey` | `crypto.key` | 总是 |
| `mysql.<id>.username` / `password` | `databases[id=<id>].user` / `password` | 每个已配置的数据库，当前为 `mysql.bkauth` |
| `redis.default.password` | `redis[*].password` | 总是 |
| `redis.default.sentinel_password` | `redis[id=sentinel].sentinelPassword` | 配置了 sentinel 模式的 Redis 时 |

注意事项：

- **`encryption.cryptoKey` 必须沿用现有的 `crypto.key`**。更换该值会导致数据库中已加密的 app secret 全部无法解密。
- `app_code` 不会被读取，`appCode` 仍来自配置文件；保留该字段只为与其他组件的信封结构一致。
- 信封中未被消费的字段会被忽略，允许多个组件共用同一份信封。
- 使用无密码 Redis 的部署不能直接开启 KMS 模式。
- `accessKeys`（部署时初始化的内置应用密钥）属于一次性初始化数据，不纳入信封。

## 轮换

应用只在进程启动时解密一次，不提供热更新。更新私钥或信封后，需要重建相关 Pod（服务、迁移及初始化 Job）。

## 测试

```bash
cd src/bkauth
go test ./pkg/config/ -count=1 -v
```

测试在运行时生成 RSA 密钥，按 bk-kms-sdk 的混合加密格式（RSA-OAEP-SHA256 + AES-CBC）构造真实信封，经 SDK 解密验证，不依赖 KMS 服务或部署凭证。
