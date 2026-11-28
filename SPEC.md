# SwitchBot BLE Prometheus Exporter の仕様

Go 版は Linux の Bluetooth アダプターで SwitchBot の BLE 広告を受信し、単発スキャンの JSON と Prometheus 用メトリクスを提供する。旧 Python 版の直接受信と Prometheus 系列を引き継ぎ、SwitchBot Cloud API や Hub を介さずに機器を観測できる。

## 動作環境と配布

- Linux、利用できる Bluetooth アダプター、BlueZ、system D-Bus が必要。基準実機は Raspberry Pi Zero W Rev 1.1（ARMv6 / armhf、512 MB RAM、内蔵 Bluetooth、Raspbian GNU/Linux 13 Trixie）。
- 配布対象は Linux amd64、arm64、ARMv7、ARMv6。配布バイナリに共有ライブラリ依存はない。ARMv6 のクロスビルドは `GOOS=linux`、`GOARCH=arm`、`GOARM=6`、`CGO_ENABLED=0` を使う。
- 実行時に Go、Python、bluepy、pip、仮想環境は不要。Taskfile は `build`、`test`、`lint`、`build-linux`、`build-armv6` を提供する。Linux 向けのクロスビルド先は `dist/switchbot-exporter-linux-<対象>`。手順は [ビルドとリリース](docs/guides/release.md) を参照する。
- 同梱の systemd Unit は `/usr/local/bin/switchbot-exporter` を実行し、`/usr/local/switchbot/nametable.json` を使う。Unit の配置先は `/etc/systemd/system/switchbot-exporter.service`。初回配備時の権限は旧版と同等とし、異常終了時は systemd が再起動する。手順は [Linux 運用ガイド](docs/guides/operations.md) を参照する。

## CLI と公開データ

| 機能 | 動作 |
| --- | --- |
| `switchbot-exporter` | 名前表に登録された機器を定期スキャンし、Prometheus の `/metrics` を公開する。既定の待受は全インターフェースの 8000 番 |
| `switchbot-exporter scan` | 既定で 10 秒間スキャンし、MAC ごとの結果を JSON で返す。広告の診断用に `--raw` を指定できる |
| `switchbot-exporter version` | アプリケーション版、Git コミット、ビルド日時、Go 版を表示する |

フラグ、名前表、出力形式、終了コードは [CLI と設定の仕様](docs/SPEC/cli.md) を参照する。Prometheus は公式 Go クライアントを使用し、旧版の系列名、ラベル、Enum の 0/1 出力を維持する。スキャン監視の系列と機種ごとの公開対象は [Prometheus 出力](docs/SPEC/metrics.md) を参照する。

## BLE 受信とスキャン制御

- BlueZ D-Bus からアダプターの状態と BLE 広告を取得する。対象は Service Data（AD Type `0x16`）と Manufacturer Data（AD Type `0xff`）。受信側で識別子とペイロードを分け、デコーダーには識別子を除いたデータを渡す。
- 常駐時は既定で `hci0` を使い、開始間隔 10 秒、受信時間 5 秒で動く。スキャン中や再試行待ちに過ぎた開始枠は飛ばし、同時実行は 1 件とする。
- BLE 障害中も HTTP を維持し、1 秒から最大 60 秒までの指数バックオフで再試行する。復旧時にアダプターの電源操作や `hciconfig` によるリセットは行わない。SIGINT / SIGTERM 時は BLE を止めてから HTTP を終了する。

対象識別子、通知順序、再試行の詳細は [BLE 受信とスキャン制御](docs/SPEC/ble.md) を参照する。

## デコードと状態保持

Service Data の先頭バイトを `0x7f` でマスクして機種を判定する。Manufacturer Data が先着した場合は MAC 単位で保留し、機種確定後に処理する。保留件数には上限があり、適用後または期限切れで破棄する。TTL の既定値は 30 秒。短いペイロードは配列にアクセスする前に検出し、該当データを破棄して処理を続ける。

機種判定、温度変換、状態文字列などは旧版との互換性を維持する。Curtain 3 の `[` と `{` は共通デコーダーを使う。Hub Mini の値は単発スキャンに出力する。デコーダーがない既知機種は種別を識別する。

デコーダーは BLE バックエンドと独立させる。Device Store は MAC ごとに機種、RSSI、更新時刻、最後の有効データを管理する。機器を一時的に検出できなくても公開済みの値を保持する。入力長、バイト配置、保持規則は [Decoder と Device Store](docs/SPEC/protocol.md) を参照する。

## 実装構成と受入条件

[詳細仕様と実装構成](docs/SPEC/README.md) にパッケージの責務と旧版との動作差を示す。[検証・受入条件](docs/SPEC/validation.md) に初期リリースの条件を、[検証記録](docs/records/verification.md) に実施結果と未確認項目を示す。
