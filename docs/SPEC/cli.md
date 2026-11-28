# CLI と設定の仕様

コマンドの構文、既定値、名前表、標準出力と終了コードを定める。起動と配備の手順は [Linux 運用ガイド](../guides/operations.md) を参照する。

## CLI と設定

`switchbot-exporter [flags]` は常駐、`switchbot-exporter scan [flags]` は単発スキャン、`switchbot-exporter version` はビルド情報を表示する。フラグは対象コマンドの後に指定する。

| フラグ | Exporter の既定値 | `scan` の既定値 |
| --- | --- | --- |
| `--config` | `./nametable.json` | 未指定。指定時は登録 MAC で絞る |
| `--adapter` | `hci0` | `hci0` |
| `--listen` | `:8000`（全インターフェース） | — |
| `--scan-duration` | `5s` | `10s` |
| `--scan-interval` | `10s` | — |
| `--cache-ttl` | `30s` | `30s` |
| `--log-level` | `info` | `info` |
| `--raw` | — | `false` |

`:8000` のようなホスト省略形式か、`127.0.0.1:8000` のようなホストとポートを指定できる。アダプター名は `hci0` のように `hci` と数字で指定する。時間は `5s` や `1m` のような正の Go duration。ログレベルは `debug`、`info`、`warn`、`error`（大文字・小文字を区別しない）。常駐時の開始間隔と再試行の関係は [スキャン制御](ble.md#実行周期と復旧) を参照する。

`nametable.json` は 6 オクテットのコロン区切り MAC をキー、表示名を文字列の値とする JSON オブジェクト。キーには十六進表記を使い、読み込み時に小文字化する。無効な MAC、`null`、正規化後の重複、空オブジェクト、オブジェクトの後に続く JSON、指定ファイルの不在はエラー。Exporter が採用する機器は名前表に登録された機器。BLE 受信側の購読範囲は [BlueZ D-Bus](ble.md#bluez-d-bus) を参照する。

## 出力と終了コード

`scan` は成功後に JSON を 1 件出力し、検出 0 台の場合は `{}` を返す。キーは小文字の MAC アドレス。`update` は Unix 秒。機種が未確定の場合や値が未取得の場合の出力は [Device Store](protocol.md#device-store) を参照する。

```json
{
  "00:11:22:33:44:55": {
    "type": 84,
    "typeStr": "SwitchBot MeterTH (WoSensorTH)",
    "rssi": -52,
    "update": 1795364397,
    "data": {
      "battery": 88,
      "temp": 24.5,
      "hum": 50,
      "group": [],
      "alert": {"temp": "no alert", "hum": "no alert"}
    }
  }
}
```

`scan --raw` は BlueZ の観測を JSON Lines で逐次出力する。各行に `mac`、取得済みなら `rssi`、UTC の `receivedAt`、`fields` を含む。フィールドの `adType` は数値、`key` は UUID 文字列または Company ID の 4 桁十六進表記、`payloadHex` は識別子を除いた小文字の十六進文字列。Service Data の `adType` は BlueZ の UUID から表現可能な最短幅を推定した値。実際のパケットは [HCI 採取](../guides/ble-poc.md#raspberry-pi-zero-w-での確認) で照合する。共有する観測ログからは端末識別情報を除く。

| 終了コード | 意味 |
| --- | --- |
| 0 | 正常完了、`--help`、常駐時の正常なシグナル終了 |
| 1 | 単発スキャンの失敗・中断、HTTP の起動・稼働エラー、出力失敗 |
| 2 | 引数または設定エラー |

ログと診断は標準エラーへ出力する。単発スキャンが失敗した場合、通常の JSON は生成しない。raw モードの途中までの観測は標準出力に残る。

`version` はアプリケーション版、Git commit、UTC のビルド日時、Go 版を表示する。Task ビルドは版・コミット・日時を埋め込む。埋め込み指定のない開発ビルドでは `dev`、`unknown`、`unknown` が既定値。
