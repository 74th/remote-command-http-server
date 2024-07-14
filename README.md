# http経由でコマンドを単純に実行するサーバ

YAML に書いたコマンドを、HTTP経由で実行するサーバです。コマンドの標準出力、標準エラー出力を、リアルタイムに返却します。

コマンドを送信する環境と、実行する環境を分けたい場合に利用する。

```
command-server ./config.yaml -p 8080
```

```
curl --no-buffer http://command-server-host:8080/call1
```

## config

```yaml
max_concurrency: 10
cmds:
  - path: /call1
    cwd: .
    cmd:
      - go
      - run
      - ./testdata/call/main.go
      - call1
    env_file: "./testdata/call1.env"
  - path: /call2
    cmd:
      - go
      - run
      - ./call/main.go
      - call2
    cwd: ./testdata
    envs:
      ENV_VAR: env_var_vall2
  - path: /{id}/call3
    cmd:
      - go
      - run
      - ./call/main.go
      - call3-{id}
    cwd: ./testdata
    envs:
      ENV_VAR: env_var_vall2
  - path: /env
    cmd:
      - /usr/bin/env
    envs:
      ENV_VAR: env_var_vall2
```