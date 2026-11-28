# CLI と Linux での運用

クイックスタートとバイナリの選び方は [README](../../README.md)。ここでは実行時の設定と systemd への配備を扱う。

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

`:8000` のようなホスト省略形式か、`127.0.0.1:8000` のようなホストとポートを指定できる。アダプター名は `hci0` のように `hci` と数字で指定する。時間は `5s` や `1m` のような正の Go duration。ログレベルは `debug`、`info`、`warn`、`error`。同時に実行するスキャンは 1 件で、ビジー時や再試行待ちの開始枠は飛ばす。

`nametable.json` は 6 オクテットのコロン区切り MAC をキー、表示名を文字列の値とする JSON オブジェクト。起動時にキーを小文字化する。無効な MAC、`null`、正規化後の重複、空オブジェクト、オブジェクトの後に続く JSON、指定ファイルの不在はエラー。Exporter が採用する機器は名前表に登録された機器。BlueZ はアダプターで広告を受信し、Exporter が MAC で絞り込む。

## 出力と終了コード

`scan` は成功後に JSON を 1 件出力し、検出 0 台の場合は `{}` を返す。更新時刻は UNIX 秒。

```json
{
  "00:11:22:33:44:55": {
    "type": 84,
    "typeStr": "SwitchBot MeterTH (WoSensorTH)",
    "rssi": -52,
    "update": 1795364397,
    "data": {"battery": 88, "temp": 24.5, "hum": 50}
  }
}
```

`scan --raw` は BlueZ の観測を JSON Lines で逐次出力する。各行に `mac`、取得済みなら `rssi`、UTC の `receivedAt`、`fields` を含む。フィールドの `adType` は数値、`key` は UUID 文字列または Company ID の 4 桁十六進表記、`payloadHex` は識別子を除いた小文字の十六進文字列。Service Data の `adType` は BlueZ の UUID から表現可能な最短幅を推定した値。実際のパケットは [HCI 採取](ble-poc.md#raspberry-pi-zero-w-での確認) で照合する。共有する観測ログからは端末識別情報を除く。

| 終了コード | 意味 |
| --- | --- |
| 0 | 正常完了、`--help`、常駐時の正常なシグナル終了 |
| 1 | 単発スキャンの失敗・中断、HTTP の起動・稼働エラー、出力失敗 |
| 2 | 引数または設定エラー |

ログと診断は標準エラーへ出力する。単発スキャンが失敗した場合、通常の JSON は生成しない。raw モードの途中までの観測は標準出力に残る。

`version` はアプリケーション版、Git commit、UTC のビルド日時、Go 版を表示する。Task ビルドは版・コミット・日時を埋め込む。埋め込み指定のない開発ビルドでは `dev`、`unknown`、`unknown` が既定値。

## systemd で稼働させる

対象アーキテクチャの Linux バイナリと、リポジトリ内の [systemd Unit](../../systemd/switchbot-exporter.service) を用意する。次の例はリポジトリのルートで、対象のバイナリを `binary` に設定して実行する。既存の Python 版が動いている場合は先に [切替と切り戻し](#切替と切り戻し) を参照する。

```sh
binary=./dist/switchbot-exporter-linux-armv6
sudo install -d -m 0755 /usr/local/switchbot
sudo install -m 0755 "$binary" /usr/local/bin/switchbot-exporter
sudo install -m 0644 systemd/switchbot-exporter.service /etc/systemd/system/switchbot-exporter.service
```

新規配備で設定を置く場合は `sudo install -m 0600 ./nametable.json /usr/local/switchbot/nametable.json` を実行する。既存の名前表がある場合はその内容を確認して使う。Unit は `WorkingDirectory=/usr/local/switchbot` から root で実行し、BlueZ 起動後に起動する。Unit の導入後に起動を許可する。

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now switchbot-exporter.service
systemctl status switchbot-exporter.service
sudo journalctl -u switchbot-exporter.service -n 30 --no-pager
```

プロセス異常終了後は systemd が 5 秒後に再起動する。SIGINT / SIGTERM ではスキャンをキャンセルして後始末を行い、その後 HTTP を最大 5 秒かけて終了する。Unit の停止上限は 15 秒。BLE の一時障害では HTTP を維持し、[スキャン監視](metrics.md#スキャン監視) に結果を反映する。[systemctl リファレンス](https://www.freedesktop.org/software/systemd/man/latest/systemctl.html)

### 起動後の確認

- `curl -fsS http://127.0.0.1:8000/metrics` で HTTP が応答すること。
- 1 回目のスキャンが終わった後に `switchbot_scan_success` が 1 になり、`switchbot_last_success_timestamp` が更新されること。
- 対象機器が近くにある場合は `switchbot_devices_seen` と機器別の `switchbot_update` を確認する。HTTP 応答があっても、スキャン失敗や未検出は起こり得る。

### よくある症状

| 症状 | 確認箇所 |
| --- | --- |
| 起動直後に終了する | ログに記載された設定パス、JSON の形式、MAC の重複、待受ポートの使用状況 |
| `switchbot_scan_success` が 0 のまま | BlueZ の稼働、`bluetoothctl list` / `bluetoothctl show` のアダプターと電源状態、D-Bus に接続できる実行ユーザー |
| スキャン成功で機器別の系列が出ない | `scan` で取得した MAC と名前表の一致、機器の距離、[公開対象の機種](metrics.md#系列とラベル) |
| 値は表示されるが鮮度が低い | `switchbot_update` と `switchbot_last_success_timestamp`。前者は最後にデコードできた測定、後者は最後に成功したスキャン |

## 切替と切り戻し

1. `systemctl show switchbot-exporter.service -p FragmentPath -p DropInPaths -p WorkingDirectory` で旧サービスの設定を確認する。旧 Unit・drop-in・Python スクリプト・名前表・置き換える既存バイナリを退避する。旧版の名前表は旧 Unit の作業ディレクトリも確認し、Go 版へ引き継ぐ。バックアップ先と旧 `/metrics` の取得時刻を記録する。
2. 同じアダプターを使う場合は旧サービスを停止した上で、Go 版を一時ポート（例 `127.0.0.1:18000`）で起動する。別アダプターを使える場合は並行比較できる。Prometheus 系列の厳密な比較には同じ入力を再生する [互換テスト](verification.md#検証項目) を使う。
3. Go 版と Unit を [systemd の配備手順](#systemd-で稼働させる) に沿って配置する。Unit を再読み込みしてサービスを起動し、`active`・`enabled`、ポート 8000 の応答、2 回以上のスキャン成功と機器別系列を確認する。旧版の `switchbot_update` が古い場合は、双方に存在する機器の系列名とラベルを比較し、照合できた台数を記録する。
4. `sudo systemctl restart switchbot-exporter.service` で再起動し、PID が変わり、最終成功時刻と機器別系列が再び更新されることを確認する。Grafana・PromQL の表示、Bluetooth 障害からの復旧、長時間稼働は [初期リリースの確認項目](verification.md#初期リリースの確認項目) に沿って記録する。
5. 起動できない、主要系列が欠ける、復旧しない場合は Go 版を停止し、退避した旧 Unit・drop-in・実行ファイルを元のパスへ戻す。`sudo systemctl daemon-reload` と `sudo systemctl start switchbot-exporter.service` を実行し、旧 `/metrics` の更新を確認する。名前表や Prometheus の既存時系列に形式変換は不要。

Pi Zero W での切替と再起動の測定結果は [本番切替の記録](verification.md#raspberry-pi-zero-w-への本番切替) を参照する。
