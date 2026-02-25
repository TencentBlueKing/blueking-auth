debug: true

# enableMultiTenantMode: false
enableMultiTenantMode: true

server:
  host: 127.0.0.1
  port: 9000

  readTimeout: 60
  writeTimeout: 60
  idleTimeout: 180

sentry:
  enable: false
  dsn: ""

pprofPassword: "DebugModel@bk"


crypto:
  # contains letters(a-z, A-Z), numbers(0-9), length should be 32 bit
  key: "tR9TnGQM8WnF1qwjjGSVE0ScXrz1hKWM"
  # length should be 12 bit
  nonce: "yuitrestrtyu"

accessKeys:
  bkauth: "G3dsdftR9nGQM8WnF1qwjGSVE0ScXrz1hKWM"
  bk_paas3: "G3dsdftR9nGQM8WnF1qwjGSVE0ScXrz1hKWM"

apiAllowLists:
  - api: "manage_app"
    allowList: "bk_paas,bk_paas3"
  - api: "read_app"
    allowList: "bk_paas,bk_paas3,bk_apigateway"
  - api: "manage_access_key"
    allowList: "bk_paas,bk_paas3"
  - api: "read_access_key"
    allowList: "bk_paas,bk_paas3,bk_apigateway"
  - api: "verify_secret"
    allowList: "bk_paas,bk_paas3,bk_apigateway,bk_iam,bk_ssm"

databases:
  - id: "bkauth"
    host: "127.0.0.1"
    port: 3306
    user: "root"
    password: ""
    name: "bkauth"
    maxOpenConns: 200
    maxIdleConns: 50
    connMaxLifetimeSecond: 600

redis:
  - id: "standalone"
    addr: "localhost:6379"
    password: ""
    db: 0
    # poolSize: 400
    # minIdleConns: 200
    dialTimeout: 5
    readTimeout: 5
    writeTimeout: 5
    masterName: ""

logger:
  system:
    level: debug
    encoding: console
    writer: os
    settings: {name: stdout}
  api:
    level: info
    encoding: json
    writer: file
    settings: {name: bkauth_api.log, size: 100, backups: 10, age: 7, path: ./}
    # 敏感日志配置
    desensitization:
      ## 日志脱敏开关配置
      enabled: true
      ## 日志脱敏规则配置: key -- 日志打印 field 的 key，jsonPath -- 日志 value 需要脱敏的 json path 路径
      fields:
        - key: body
          jsonPath:
            - "bk_app_secret"
        - key: response_body
          jsonPath:
            - "bk_app_secret"
            - "data.#.bk_app_secret"
  sql:
    level: debug
    encoding: json
    writer: file
    settings: {name: bkauth_sql.log, size: 100, backups: 10, age: 7, path: ./}
  audit:
    level: info
    encoding: json
    writer: file
    settings: {name: bkauth_audit.log, size: 500, backups: 20, age: 365, path: ./}
  web:
    level: info
    encoding: json
    writer: file
    settings: {name: bkauth_web.log, size: 100, backups: 10, age: 7, path: ./}

observability:
  enable: false

  service:
    name: "bkauth"
    version: "v1.0.0"
    environment: "prod"

  exporter:
    endpoint: "http://localhost:4318"
    token: ""

  signals:
    traces:
      enable: true
      sampler:
        # 支持 always_on / always_off / traceidratio / parentbased
        type: "always_on"
        # 采样比例: type=traceidratio 或 parentbased 时生效
        ratio: 0.1
      batch:
        timeout: "5s"
        maxExportBatchSize: 512
        maxQueueSize: 2048

    metrics:
      enable: false

    logs:
      enable: false
      # 上报到观测平台的最低日志级别: debug / info / warn / error（默认 info）
      level: "info"
      # 指定哪些 logger 上报到 OTEL，可选: system / api / web / sql / audit
      loggers:
        - system

    profiling:
      enable: false
      path: "/pyroscope"
      uploadInterval: "15s"
      # Pyroscope 专用 Endpoint: 为空时继承 exporter.endpoint
      endpoint: ""
      # Pyroscope 专用 Token: 为空时继承 exporter.token
      token: ""
