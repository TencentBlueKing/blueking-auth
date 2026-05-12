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
monitoringToken: "YbqrbLCTcbZyCHS82JDqYvMpuDuM3t2Dq4iF"

crypto:
  # contains letters(a-z, A-Z), numbers(0-9), length should be 32 bit
  key: "tR9TnGQM8WnF1qwjjGSVE0ScXrz1hKWM"
  # length should be 12 bit
  nonce: "yuitrestrtyu"

accessKeys:
  bkauth: "G3dsdftR9nGQM8WnF1qwjGSVE0ScXrz1hKWM"
  bk_paas3: "G3dsdftR9nGQM8WnF1qwjGSVE0ScXrz1hKWM"

appCode: "bkauth"
appSecret: "WnF1qwjGSVE0ScXrz1hKWMG3dsdftR9nGQ"
bkAuthUrl: "https://bkauth.example.com"
bkApiUrlTmpl: "http://bkapi.example.com/api/{api_name}"
bkLoginUrl: "https://bk.example.com/login/"
# true: call login API via gateway; false: call directly (default)
bkLoginAPIViaGateway: false
# "bk_token" (default) or "bk_ticket"
bkLoginTokenName: "bk_token"

oauth:
  defaultRealmName: "blueking"
  dcrEnabled: false
  accessTokenTTL: 7200
  refreshTokenTTL: 2592000
  introspectAllowedAppCodes:
    - realmName: "blueking"
      appCode: "bk_apigateway"
    # - realmName: "bk-devops"
    #   appCode: "bk_devops_gateway"
    # - realmName: "*"
    #   appCode: "bk_super_app"
  # confidentialClientSecretExemptions:
  #   - realmName: "*"
  #     clientID: "bk_my_desktop_app"
  #   - realmName: "blueking"
  #     clientID: "bk_another_app"
  # tokenTTLOverrides:
  #   - realmName: "blueking"
  #     clientID: "*"
  #     accessTokenTTL: 3600
  #     refreshTokenTTL: 604800
  #   - realmName: "blueking"
  #     clientID: "my_special_app"
  #     accessTokenTTL: 900

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
      ## 注意: form-urlencoded 请求体会被中间件自动转为 JSON，因此 jsonPath 规则对 form 字段同样生效
      fields:
        - key: body
          jsonPath:
            # App API
            - "bk_app_secret"
            # OAuth: client credentials
            - "client_secret"
            # OAuth: authorization code / token exchange
            - "code"
            - "code_verifier"
            - "refresh_token"
            - "device_code"
            # OAuth: revoke / introspect
            - "token"
        - key: response_body
          jsonPath:
            # App API
            - "bk_app_secret"
            - "data.#.bk_app_secret"
            # OAuth: token response
            - "access_token"
            - "refresh_token"
            # OAuth: device authorization response
            - "device_code"
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

trace:
  enabled: false
  otlp:
    host: "localhost"
    port: 4318
    token: ""
    type: "http"
  serviceName: "bkauth"
  # always_on: 总是上报，前端或上游没接入时建议使用
  # parentbased_always_on: 跟随上游采样决策
  sampler: "always_on"

profiling:
  enabled: false
  pyroscope:
    host: "localhost"
    port: 4318
    token: ""
    type: "http"
    path: "/pyroscope"
  serviceName: "bkauth"
  uploadInterval: "15s"
