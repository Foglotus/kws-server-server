# 非管理员接口签名对接文档

本文档用于前端/客户端对接非管理员接口签名。

## 1. 生效范围

- 需要签名：所有非管理员接口（例如 `/realkws/api/v1/*`、`/realkws/health`、`/realkws/`）
- 不需要签名：`/realkws/admin/*`（管理员接口使用独立鉴权）
- 预检请求：`OPTIONS` 不校验签名

## 2. 签名规则

- 待签名字符串：

```text
path + timestamp
```

- 签名算法：

```text
SHA256(path + timestamp)
```

- 签名编码：十六进制小写字符串

### 字段说明

- `path`: 请求路径（不含域名、不含 query）
  - 例：`/realkws/api/v1/offline/asr`
- `timestamp`: Unix 时间戳（秒）字符串
  - 例：`1714288888`

## 3. 请求头/参数

### HTTP 接口

在请求头中传：

- `X-Timestamp: <timestamp>`
- `X-Signature: <signature>`

### WebSocket 接口

浏览器原生 WebSocket 不能自定义 Header，建议通过 query 传：

- `timestamp`
- `signature`

示例：

```text
ws://127.0.0.1:11123/realkws/api/v1/streaming/asr?timestamp=1714288888&signature=xxxx
```

## 4. 时间有效期

默认允许时间偏差：`300` 秒（可通过 `signature.max_skew_seconds` 调整）。

若客户端和服务端时间差过大，会返回 `timestamp expired`。

## 5. JavaScript 对接示例

> 推荐在可信后端计算签名，不要把 `secret` 暴露在公开前端页面。

### 5.1 浏览器侧（Web Crypto）

```javascript
async function sha256Hex(message) {
  const encoder = new TextEncoder();
  const data = encoder.encode(message);
  const digest = await crypto.subtle.digest('SHA-256', data);
  return Array.from(new Uint8Array(digest))
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

async function signedFetch(baseURL, path, options = {}) {
  const timestamp = Math.floor(Date.now() / 1000).toString();
  const signature = await sha256Hex(path + timestamp);

  const headers = {
    ...(options.headers || {}),
    'X-Timestamp': timestamp,
    'X-Signature': signature,
  };

  return fetch(baseURL + path, {
    ...options,
    headers,
  });
}

// 调用示例
// const resp = await signedFetch('http://127.0.0.1:11123', '/realkws/api/v1/stats', { method: 'GET' });
```

### 5.2 WebSocket（query 参数）

```javascript
async function buildSignedWSURL(baseWS, path) {
  const timestamp = Math.floor(Date.now() / 1000).toString();
  const signature = await sha256Hex(path + timestamp);
  return `${baseWS}${path}?timestamp=${encodeURIComponent(timestamp)}&signature=${encodeURIComponent(signature)}`;
}

// const wsUrl = await buildSignedWSURL('ws://127.0.0.1:11123', '/realkws/api/v1/streaming/asr');
// const ws = new WebSocket(wsUrl);
```

## 6. Node.js 对接示例

```javascript
import crypto from 'crypto';

function signPath(path, timestamp) {
  return crypto
    .createHash('sha256')
    .update(path + timestamp)
    .digest('hex');
}

function buildSignedHeaders(path) {
  const timestamp = Math.floor(Date.now() / 1000).toString();
  const signature = signPath(path, timestamp);
  return {
    'X-Timestamp': timestamp,
    'X-Signature': signature,
  };
}
```

## 7. 常见错误

- `missing signature or timestamp`
  - 未携带 `X-Timestamp` / `X-Signature`（或 WebSocket query 参数）
- `invalid timestamp`
  - 时间戳非数字
- `timestamp expired`
  - 时间戳与服务端偏差超过允许窗口
- `invalid signature`
  - 签名算法、路径或时间戳不一致

## 8. 服务端配置示例

```yaml
signature:
  enabled: true
  max_skew_seconds: 300
```
