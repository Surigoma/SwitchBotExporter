# SwitchBot BLE Prometheus Exporter

SwitchBot の BLE Advertisement を BlueZ 経由で受信し、Prometheus の `/metrics` に公開する Linux 向け Exporter です。実行環境には Bluetooth アダプター、BlueZ、system D-Bus が必要です。SwitchBot Cloud API や Hub を経由せずに受信できます。

## 使い始める

### 1. バイナリを用意する

`uname -m` で環境を確認し、[GitHub Releases](https://github.com/Surigoma/SwitchBotExporter/releases) から環境に対応するアーカイブと `SHA256SUMS` を取得します。

| 環境 | ファイル名の末尾 |
| --- | --- |
| `x86_64` | `linux-amd64.tar.gz` |
| `aarch64`（64 ビット Linux） | `linux-arm64.tar.gz` |
| `armv7l`（32 ビット Linux） | `linux-armv7.tar.gz` |
| `armv6l`（Raspberry Pi Zero W など） | `linux-armv6.tar.gz` |

次の例は Raspberry Pi Zero W 用です。ほかの機種ではファイル名を対応するものに置き換えます。

```sh
sha256sum --ignore-missing -c SHA256SUMS
tar -xzf switchbot-exporter-linux-armv6.tar.gz
./switchbot-exporter-linux-armv6 version
```

公開済みリリースがなければ、ソースからビルドできます。ビルドには `go.mod` が指定する Go 1.25 以上と [Task](https://taskfile.dev/docs/installation) が必要です。

```sh
task build-linux ARCH=amd64
task build-linux ARCH=arm64
task build-linux ARCH=arm ARM_VERSION=7
task build-armv6
```

対象機器に `dist/switchbot-exporter-linux-<対象>` を転送します。ビルドするホストと実行する Linux ホストが同じ場合は `task build` で `dist/switchbot-exporter` を生成できます。ほかのアーキテクチャの指定方法は [Linux の配布形式](docs/SPEC/release.md#配布形式) を参照してください。

### 2. 機器を見つける

Linux ホストで単発スキャンを行います。既定のアダプターは `hci0`、スキャン時間は 10 秒です。結果の JSON のキーが機器の MAC アドレスです。

```sh
./switchbot-exporter-linux-armv6 scan
```

公開する機器だけを `nametable.json` に登録します。MAC は例示値を置き換え、値には Prometheus の `name` ラベルにしたい名前を指定します。

```json
{
  "00:11:22:33:44:55": "example-meter"
}
```

### 3. メトリクスを確認する

Linux ホスト上でまずループバックに待ち受けて動作を確認します。既存サービスが 8000 番を使っている環境では、比較には別ポートを使ってください。

```sh
./switchbot-exporter-linux-armv6 --config ./nametable.json --listen 127.0.0.1:8000
```

別のターミナルで、初回のスキャン完了後に以下を実行します。`switchbot_scan_success 1` は最後のスキャンが成功したことを示します。対象機器からデータを受信すると、機器別系列も表示されます。

```sh
curl -fsS http://127.0.0.1:8000/metrics
```

`--listen` を省略した場合は全インターフェースの 8000 番で待ち受けます。Prometheus からは Linux ホストのアドレスと待受ポートをターゲットにし、`/metrics` を取得します。

## 目的別ガイド

| やりたいこと | 参照先 |
| --- | --- |
| 実行オプションや JSON 形式を調べる | [CLI と設定](docs/SPEC/operations.md#cli-と設定) |
| systemd 配備、Python 版からの移行、切り戻し | [Pi での運用](docs/SPEC/operations.md#systemd-で稼働させる) |
| 対応機器、デコードや Prometheus の互換仕様を調べる | [仕様書](SPEC.md) |
| 機種を追加する、テストや CI を実行する | [開発と検証](docs/SPEC/verification.md) |
| GitHub 上でリリースを作成・再作成する | [リリース作業](docs/SPEC/release.md) |
| 実機で確認済みの項目を調べる | [検証結果](docs/SPEC/verification.md#確認済みの結果) |
