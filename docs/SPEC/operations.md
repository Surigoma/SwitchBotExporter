# CLI・検証・移行

上位仕様の [§8–16](../../SPEC.md) に対応する。旧版では [単発スキャナー](../../.orig/switchbot_scanner.py) が全デバイスを 10 秒取得し、Exporter は [nametable 形式](../../.orig/nametable.json) に登録された MAC を対象にする。

## CLI と設定

`switchbot-exporter [flags]` は Exporter、`switchbot-exporter scan [flags]` は一回のスキャン、`switchbot-exporter version` はビルド情報を標準出力に表示する。フラグはコマンドの後に置く。Go の `flag` と `slog` を使う。

| フラグ | Exporter 既定値 | scan 既定値 |
| --- | --- | --- |
| `--config` | `./nametable.json` | 未指定。指定時は登録 MAC で絞る |
| `--adapter` | `hci0` | `hci0` |
| `--listen` | `:8000` | — |
| `--scan-duration` | `5s` | `10s` |
| `--scan-interval` | `10s` | — |
| `--cache-ttl` | `30s` | `30s` |
| `--log-level` | `info` | `info` |
| `--raw` | — | false |

`nametable.json` は JSON オブジェクト 1 個で、キーは 6 オクテットのコロン区切り MAC、値は文字列。起動時に読み込み、キーは小文字に正規化する。無効な MAC、null 値、重複キー（大小文字差を含む）、空オブジェクト、余分な JSON、ファイル不在は設定エラー。表示名は文字列をそのまま利用する。 `--scan-duration`、`--scan-interval`、`--cache-ttl` は正の Go duration とし、`--log-level` は `debug/info/warn/error`。未登録のフラグや余分な位置引数は引数エラー。設定は起動時に確定する。

`scan` は一回の成功完了後、`{"00:11:22:33:44:55":{"type":84,"typeStr":"SwitchBot MeterTH (WoSensorTH)","rssi":-52,"update":1795364397,"data":{"battery":88,"temp":24.5,"hum":50}}}` の形を出力する。検出 0 件は `{}`。 `--raw` ではデコード結果の代わりに BlueZ が通知した Service Data / Manufacturer Data の観測を JSON Lines で逐次出力する。各行は `mac`、取得できた場合の `rssi`、RFC 3339 Nano の `receivedAt`、`fields` を持つ。 `fields` の各要素は `adType`（数値）、`key`（UUID 文字列または 4 桁の Company ID の十六進文字列）、`payloadHex`（識別子を除いた小文字の十六進文字列）。対象 ID への絞り込み前の通知を出力し、MAC 設定が指定されていれば適用する。出力には端末識別情報を含むため、配布するフィクスチャから識別情報を除く。原始 HCI パケットの比較には実機で別途採取したキャプチャを用いる。

正常完了は終了コード 0、引数・設定エラーは 2、単発スキャン失敗や HTTP の起動・稼働エラーは 1。 `scan` の通常 JSON は失敗時に出力せず、`--raw` は失敗までに出力した観測が残る。ログと診断は標準エラー。Exporter が SIGINT/SIGTERM で正常に停止した場合は 0。単発スキャンが完了前に中断された場合は 1。

`version` はアプリケーション版、Git commit、UTC の Build Date、`runtime.Version()` の 4 項目を出す。配布ビルドでは build 時に各値を埋め込み、未指定の開発ビルドでは版を `dev`、commit・日付を `unknown` とする。

## 実装順序と受入条件

| 順序 | 変更箇所と成果物 | 次に進む条件 |
| --- | --- | --- |
| 1. BLE PoC | `internal/ble` と実機確認用エントリーポイント | Pi で [BLE PoC の合格条件](ble.md#poc-の合格条件) を記録し、Backend と正規化を確定する |
| 2. Decoder | `internal/protocol`、`internal/device`、`testdata/advertisements/` | 全対応型の正常値、全フィールドの境界、順序と TTL を自動テストで確認する |
| 3. Exporter | `internal/metrics`、`internal/app` の HTTP 経路 | Python 版の系列と比較し、対象機種・ラベル・状態・値・系列数が期待どおり |
| 4. 常駐化と CLI | `cmd/switchbot-exporter`、`internal/config`、`internal/app` の実行制御 | 無線障害中の scrape、復旧、半端な受信・停止・設定エラーを検証する |
| 5. 配布と移行 | `Taskfile.yml`、systemd Unit、ルートの `README.md` | ARMv6 バイナリ、Pi 実機の再起動と連続動作、既存クエリ、切り戻しを確認する |

各段階で実行環境・テスト結果・再現手順・未解決事項を記録する。最初の実装成果物は BLE PoC とする。

## 自動テストと実機検証

| 対象 | 入力・条件 | 期待値 |
| --- | --- | --- |
| 正規化 | 対象 UUID / Company ID、非対象 ID、無関係の Adapter、初期スナップショット、変更通知、RSSI なし | 対象の値に識別子の二重除去がなく、古い値や非対象通知を新規測定として扱わない |
| デコード | 全型の正常・最小長・1 バイト不足、空、未知型、有効フラグ付き ID、長いデータ、全 Enum、符号・小数・Fahrenheit、0/false | [デコード表](protocol.md#バイト配置) と一致し、長さ不足はエラー。Contact の拡張時刻と `-1`、Hub Mini の 40 bit、Curtain 3 の 2 種別も確認する |
| Store | S 先行、M 先行、同一イベント、重複、複数 MAC、TTL 直前と境界、上限越え、不正 M と正常 M の混在、型変更、RSSI 単独 | MAC と型を取り違えず、有効値のみ保持。時刻が再生で逆行せず、欠けた値を合成しない |
| 互換性 | 同一の受信列、固定時刻、同一の名前設定で Python と Go に入力 | Prometheus テキストを解析し、`switchbot_*` の既存系列名、型、ラベル、値、系列数を比較する。小数は誤差を許容する。公開時刻や部分データなど設計で変更したケースは差分用の期待値を持つ |
| スケジューラ | フェイク Backend と制御可能な時計で通常、遅延、Start/Stop 失敗、接続断、キャンセルを再現 | 5/10 秒の既定動作、重複なし、指数バックオフと成功時リセット、監視値、後始末の順序を確認する |
| HTTP / CLI | `httptest`、並行 scrape、未設定と設定不備、部分データ、空結果、バージョン出力 | 状態切替の途中を返さず、JSON・終了コード・標準エラーの契約を満たす |
| Pi 実機 | systemd 起動・停止・再起動、Bluetooth 再起動、Adapter OFF/ON、HTTP の継続、長時間の観測 | 自動復旧し、停止期限内でセッションを解放する。連続稼働の初回確認は 24 時間を目安とし、RSS と試行回数、取得漏れ、監視値を記録する |

実機フィクスチャには機種、信号の種類、取得方法、想定結果、識別情報の匿名化有無を添える。架空データは合成と明示する。Python 比較は旧クライアントの版を記録し、実機の BLE を使わない比較では delegate へ同じ入力を再生する。Go の単体テストは D-Bus や実機を使わずに実行できるようにする。

### 実機スモークテスト結果

Raspberry Pi Zero W 上で ARMv6 バイナリの `version` と `scan` を実行し、単発スキャンで 8 台の値をデコードした。ループバックの一時ポートでは、2 回連続のスキャン成功と失敗 0 件を確認し、`/metrics` から機器別の 41 系列を取得した。`SIGTERM` で正常終了した。存在しないアダプターを指定した場合も `/metrics` が応答し、エラーカウンターが増えた。検証の前後で既存の Python Exporter の起動時刻と再起動回数は変化しなかった。

実際の BlueZ 再起動・アダプター電源断からの復旧、systemd Unit を用いた切替、長時間稼働は移行時の確認項目とする。

## ビルドと systemd

`task build` は通常ビルド、`task test` は `go test ./...`、`task lint` は `gofmt` の差分確認と `go vet ./...`、`task build-armv6` は `GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0` で `dist/switchbot-exporter-linux-armv6` を生成する。並行処理は対応するホストで `go test -race ./...` を実行する。Go の ARMv6 対応と依存モジュールの Go 最小版を確認後、採用版を記録して `go.mod` と `go.sum` を固定する。成果物は `file`、`readelf -l` と `readelf -d` で 32 bit ARM ELF、動的インタープリターと共有ライブラリ依存の有無を調べ、Pi 上で `version`、`scan`、Exporter を実行して合格とする。[Go on ARM](https://go.dev/wiki/GoArm)。

`systemd/switchbot-exporter.service` には以下を使用する。初回は旧 Unit と同等の root 権限で BlueZ system bus へ接続し、実機で実行条件を確認する。

```ini
[Unit]
Description=SwitchBot BLE Prometheus Exporter
After=bluetooth.service
Wants=bluetooth.service

[Service]
Type=simple
User=root
WorkingDirectory=/usr/local/switchbot
ExecStart=/usr/local/bin/switchbot-exporter
Restart=always
RestartSec=5
TimeoutStopSec=15

[Install]
WantedBy=multi-user.target
```

SIGINT/SIGTERM でスキャンをキャンセルして後始末を待ち、`http.Server.Shutdown` に初期値 5 秒の期限を設定する。HTTP の期限を超えた接続は閉じる。起動時のリスン失敗はプロセスエラーとして扱い、稼働中の BLE 障害は HTTP を維持したまま監視値に反映する。systemd の停止上限に収まることを実機で測定する。

## 切替と切り戻し

1. 配備済みの Python スクリプト、Unit、`nametable.json`、旧 `/metrics` 応答を控える。Go バイナリを `/usr/local/bin/switchbot-exporter` に配置して版を確認する。
2. 可能なら別アダプターで新旧を比較する。同一アダプターしかない場合は順番に停止・起動し、Go は比較用ポート（例 `:18000`）で確認する。旧版の Bluetooth リセットとポート競合を避ける。同じ受信データの厳密比較には再生テストを使う。
3. `/etc/systemd/system/switchbot-exporter.service` を切り替え、`systemctl daemon-reload` 後に起動する。既存設定のパスは `/usr/local/switchbot/nametable.json`。実機でポート 8000、scrape、既存 Grafana と PromQL、スキャン成功時刻、BLE 障害からの復旧を確認する。
4. 起動失敗、主要系列の欠落、BLE 復旧失敗を確認したら Go 版を停止し、控えた旧 Unit と実行パスを復元する。 `systemctl daemon-reload` 後に旧版を起動し、ポート 8000 と既存系列の更新を確認する。名前表と Prometheus 側の既存時系列に変換作業は発生しない。
