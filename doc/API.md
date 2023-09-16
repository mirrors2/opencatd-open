# API Documentation

## 用户

### 初始化用户

- URL: `/1/users/init`
- Method: `POST`
- Description: 初始化用户

Resp:
```
{
  "id" : 1,
  "updatedAt" : "2023-05-28T18:47:19+08:00",
  "name" : "root",
  "token" : "df5982d6-d8fb-4eef-a513-24d0831f61ad",
  "createdAt" : "2023-05-28T18:47:19+08:00"
}
```

### 获取当前用户信息

- URL: `/1/me`
- Method: `GET`
- Description: 获取当前用户信息
- Requires authentication: Yes
- Headers:
    - Authorization: Bearer {token}

Resp:
```
{
  "id" : 1,
  "updatedAt" : "2023-05-28T18:47:19+08:00",
  "name" : "root",
  "token" : "df5982d6-d8fb-4eef-a513-24d0831f61ad",
  "createdAt" : "2023-05-28T18:47:19+08:00"
}
```

### 获取所有用户信息

- URL: `/1/users`
- Method: `GET`
- Description: 获取所有用户信息
- Requires authentication: Yes
- Headers:
    - Authorization: Bearer {token}

Resp:
```
[
  {
    "createdAt" : "2023-05-28T18:47:19.711027498+08:00",
    "id" : 1,
    "updatedAt" : "2023-05-28T18:47:19.711027498+08:00",
    "IsDelete" : false,
    "name" : "root",
    "token" : "df5982d6-d8fb-4eef-a513-24d0831f61ad"
  },
  {
    "createdAt" : "2023-05-28T18:48:29.018428441+08:00",
    "id" : 2,
    "updatedAt" : "2023-05-28T18:48:29.018428441+08:00",
    "IsDelete" : false,
    "name" : "u1",
    "token" : "6ac4bd1a-18a6-4c25-922f-db689a299e38"
  }
]
```

### 添加用户

- URL: `/1/users`
- Method: `POST`
- Description: 添加用户
- Headers:
    - Authorization: Bearer {token}

Req:
```
{
  "name" : "u1"
}
```

Resp:
```
{
  "createdAt" : "2023-05-28T18:48:29.018428441+08:00",
  "id" : 2,
  "updatedAt" : "2023-05-28T18:48:29.018428441+08:00",
  "IsDelete" : false,
  "name" : "u1",
  "token" : "6ac4bd1a-18a6-4c25-922f-db689a299e38"
}
```

### 删除用户

- URL: `/1/users/:id`
- Method: `DELETE`
- Description: 删除用户
- Headers:
    - Authorization: Bearer {token}

Resp:
```
{
  "message" : "ok"
}
```

### 重置用户 Token

- URL: `/1/users/:id/reset`
- URL: `/1/users/:id/reset?token={new user token}`
- Method: `POST`
- Description: 重置用户 Token 默认生成新 Token 也可以指定
- Headers:
    - Authorization: Bearer {token}

Resp:
```
{
  "createdAt" : "2023-05-28T19:33:54.83935763+08:00",
  "id" : 2,
  "updatedAt" : "2023-05-28T19:34:33.387843151+08:00",
  "IsDelete" : false,
  "name" : "u2",
  "token" : "881a30d2-2fc8-4758-a07e-7d9ad5f34266"
}
```
## Key

### 获取所有 Key

- URL: `/1/keys`
- Method: `GET`
- Description: 获取所有 Key
- Headers:
    - Authorization: Bearer {token}

```
[
  {
    "id" : 1,
    "key" : "sk-zsbdzsbdzsbdzsbdzsbdzsbdzsbd",
    "createdAt" : "2023-05-28T18:47:49.936644953+08:00",
    "updatedAt" : "2023-05-28T18:47:49.936644953+08:00",
    "name" : "key",
    "ApiType" : "openai"
  },
  {
    "id" : 2,
    "key" : "1234567890qwertyuiopasdfghjklzxcvbnm",
    "createdAt" : "2023-05-28T18:48:18.548627422+08:00",
    "updatedAt" : "2023-05-28T18:48:18.548627422+08:00",
    "name" : "key2",
    "ApiType" : "openai"
  }
]
```

### 添加 Key

- URL: `/1/keys`
- Method: `POST`
- Description: 添加 Key
- Headers:
    - Authorization: Bearer {token}

Req:
```
{
  "key" : "sk-zsbdzsbdzsbdzsbdzsbdzsbdzsbd",
  "name" : "key",
  "api_type": "openai",
  "endpoint": ""

}
```
api_type:不传的话默认为“openai”;当前可选值[openai,azure,claude]
endpoint: 当 api_type 为 azure_openai时传入（目前暂未使用）

Resp:
```
{
  "id" : 1,
  "key" : "sk-zsbdzsbdzsbdzsbdzsbdzsbdzsbd",
  "createdAt" : "2023-05-28T18:47:49.936644953+08:00",
  "updatedAt" : "2023-05-28T18:47:49.936644953+08:00",
  "name" : "key",
  "ApiType" : "openai"
}
```

### 删除 Key

- URL: `/1/keys/:id`
- Method: `DELETE`
- Description: 删除 Key
- Headers:
    - Authorization: Bearer {token}

Resp:

```
{
  "message" : "ok"
}
```

## Usages

### 获取用量信息

- URL: `/1/usages?from=2023-03-18&to=2023-04-18`
- Method: `GET`
- Description: 获取用量信息
- Headers:
    - Authorization: Bearer {token}

Resp:
```
[
  {
    "cost" : "0.000110",
    "userId" : 1,
    "totalUnit" : 55
  },
  {
    "cost" : "0.000110",
    "userId" : 2,
    "totalUnit" : 55
  }
]
```

## Whisper接口
### 与openai一致
