# SwitchBot BLE Prometheus Exporter の仕様

対象は Go 版の `switchbot-exporter`。この文書では機能要件を定め、バイト配置や運用手順は各詳細文書で扱う。実装の構成は [詳細設計の索引](docs/SPEC/README.md) を参照する。

## 1. プロジェクト概要

Python 版の BLE スキャン、SwitchBot プロトコルのデコード、Prometheus Exporter を Go へ移植する。BLE の直接受信と既存の Prometheus 系列を維持し、Python・bluepy への実行時依存を解消する。クラウドや SwitchBot Hub を介さずに機器を観測できること、Bluetooth 障害後に自動復旧できること、ARMv6 用の単一バイナリとして配布できることを要件とする。

## 2. 動作環境

実行環境は Linux、Bluetooth アダプター、BlueZ、system D-Bus。基準実機は Raspberry Pi Zero W Rev 1.1（ARMv6 / armhf、512 MB RAM、内蔵 Bluetooth、Raspbian GNU/Linux 13 Trixie）。Linux amd64・arm64・ARMv7・ARMv6 を配布対象とする。ARMv6 のクロスビルドには `GOOS=linux`、`GOARCH=arm`、`GOARM=6`、`CGO_ENABLED=0` を指定する。配布バイナリは共有ライブラリに依存しない。

## 3. 現行システムの仕様

移植元は Python 版の `switchbot.py`（受信・デコード）、`switchbot_scanner.py`（単発 JSON）、`switchbot_exporter.py`（定期スキャン・メトリクス）。Go 版はこれらを単一コマンドに統合する。

- 対象 AD Type は Service Data の `0x16` と Manufacturer Data の `0xff`。対象識別子とバイト順は [ペイロード正規化](docs/SPEC/ble.md#ペイロード正規化) に従う。
- Service Data の先頭バイトを `0x7f` でマスクして機種を判定する。Manufacturer Data が先着しても、MAC 単位で保留し、機種確定後に処理する。
- 保留データは件数を制限し、デコード後または期限切れで破棄する。TTL の既定値は 30 秒で、変更できる。

## 4. 対応デバイスと取得情報

[種別と最小長](docs/SPEC/protocol.md#種別表と入力長) と [バイト配置](docs/SPEC/protocol.md#バイト配置) を適用する。既存の温度の符号・小数・Fahrenheit 変換、バッテリーのマスク、状態文字列、Contact Sensor のカウンターと時刻拡張を保持する。Curtain 3 の `[` と `{` は同じデコーダーを使う。Hub Mini の取得値は単発スキャン向けとし、デコーダーのない機種は種別を識別する。短いペイロードは配列アクセス前に検出し、当該データを拒否して処理を継続する。

## 5. システムアーキテクチャ

BlueZ D-Bus でアダプターを検出し、スキャン開始・停止、MAC・RSSI・広告の取得、アダプター状態の監視を行う。受信側が識別子を正規化し、識別子を除いたペイロードをデコーダーへ渡す。デコーダーは BLE バックエンドと独立させる。

MAC 単位の Device Store は機種・RSSI・更新時刻・最後の有効なデータを管理する。機器が一時的に見えなくなっても公開済みの値を保持する。受信・正規化は [BLE 受信](docs/SPEC/ble.md)、保存規則は [Device Store](docs/SPEC/protocol.md#device-store) を参照する。

## 6. Prometheus Exporter

Prometheus 公式 Go クライアントを使い、既定では全インターフェースの 8000 番で `/metrics` を公開する。既存のメトリクス名、ラベル名・値、Enum の系列と 0/1 の意味を保持する。更新時刻は UNIX 秒。バッテリー未取得時の系列は生成しない。スキャン成功状態、エラー数、所要時間、検出台数、最終成功時刻を監視用に出す。[メトリクスと互換性](docs/SPEC/metrics.md)

## 7. スキャン制御

常駐時は 5 秒間のスキャンを開始間隔 10 秒で行う。アダプターの既定値は `hci0`。スキャンを重複させず、実行中や再試行待ちで経過した枠は飛ばす。BLE 障害中も HTTP を維持し、1 秒から最大 60 秒までの指数バックオフを用いて再試行する。アダプター電源の強制操作や `hciconfig` によるリセットは行わない。SIGINT / SIGTERM 時は BLE 側を止めてから HTTP を終了する。[スキャン周期と復旧](docs/SPEC/ble.md#実行周期失敗時の動作)

## 8. 設定ファイル

`nametable.json` は MAC と表示名を対応させる JSON オブジェクト。MAC の大文字・小文字を正規化し、Exporter は登録済み機器の値を採用する。設定不在や登録件数 0 件は起動エラーとする。設定パス、アダプター、待受、スキャン時間・間隔、TTL、ログレベルを指定できる。[フラグと既定値](docs/SPEC/operations.md#cli-と設定)

## 9. CLI仕様

`switchbot-exporter` は常駐、`switchbot-exporter scan` は既定 10 秒の単発 JSON スキャン、`switchbot-exporter version` はアプリケーション版、Git commit、ビルド日時、Go 版を表示する。単発スキャンには広告の診断用出力も用意する。[CLI 出力の形式](docs/SPEC/operations.md#出力と終了コード)

## 10. ソースコード構成

入口は `cmd/switchbot-exporter`。受信、共通プロトコル、機種別デコード、状態管理、メトリクス、設定、実行制御の責務は分離する。機種固有の処理は `internal/models/<機種>/` に置き、種別表へ登録する。[パッケージと責務](docs/SPEC/README.md#実行経路とパッケージ)

## 11. テスト仕様

機種別デコードと境界値、イベント順序とキャッシュ、Python 版の Prometheus 互換性、BLE と HTTP の結合動作、復旧と正常終了、ARMv6 ビルドと実機動作を検証する。実機から取得したフィクスチャは由来を記録し、合成入力と区別する。[検証項目](docs/SPEC/verification.md#検証項目)

## 12. ビルド・配布

Taskfile で `build`、`test`、`lint`、`build-linux`、`build-armv6` を提供する。クロスビルドしたバイナリは `dist/switchbot-exporter-linux-<対象>` に生成する。実行時には Go、Python、pip、仮想環境を必要としない。[配布と公開手順](docs/SPEC/release.md)

## 13. systemd

`/usr/local/bin/switchbot-exporter`、`/usr/local/switchbot/nametable.json`、`/etc/systemd/system/switchbot-exporter.service` に配備できること。初回の配備は旧版と同等の権限を使い、異常終了後は systemd が再起動する。[配備・移行・切り戻し](docs/SPEC/operations.md#systemd-で稼働させる)

## 14. 実装マイルストーン

BLE PoC、Decoder、Prometheus Exporter、常駐化、実機移行の順に進める。実機 PoC で BlueZ D-Bus の採用を決めた。各段階の検証と残作業は [開発と検証](docs/SPEC/verification.md#移植時の実装順序) を参照する。

## 15. 完成条件

[初期リリースの確認項目](docs/SPEC/verification.md#初期リリースの確認項目) をすべて満たすこと。合否は合成テスト、実機スモークテスト、運用上の未確認事項を区別して記録する。

## 16. Codexへの実装指示

移植時は BLE PoC を最初の実装成果物とする。[移植時の実装順序](docs/SPEC/verification.md#移植時の実装順序) に沿って結果と未解決事項を報告する。プロトコルの判断は元実装または実機の証拠に基づける。互換性への意図的な変更を明記し、性能最適化は実測を根拠とする。
