# 詳細仕様と実装構成

Go 版の機能要件は [上位仕様](../../SPEC.md) にまとめる。詳細仕様の互換性は移植元の Python 実装に基づく。互換テストの入力と期待値は [合成フィクスチャ](../../testdata/advertisements/README.md) に収録する。

| 文書 | 確認できること |
| --- | --- |
| [CLI と設定](cli.md) | フラグ、名前表、標準出力、終了コード |
| [BLE 受信とスキャン制御](ble.md) | BlueZ D-Bus、正規化、スキャン周期と復旧 |
| [Decoder と Device Store](protocol.md) | 機種 ID、バイト配置、結合と状態保持 |
| [Prometheus 出力](metrics.md) | 系列名、ラベル、Enum、監視メトリクス |
| [検証・受入条件](validation.md) | テストの観点、初期リリースと BLE 受信方式の評価条件 |

## 実行経路とパッケージ

```text
cmd/switchbot-exporter
  ├─ config: 起動引数と nametable.json を検証
  └─ app: スキャンと HTTP のライフサイクルを管理
       ├─ ble: BlueZ の通知 → 正規化された Advertisement
       ├─ device: MAC 単位で models を呼び出し受信状態を更新
       │    ├─ models: 種別表、機種別のデコードとメトリクス変換
       │    └─ protocol: 正規化モデルと共通のバイト演算
       └─ metrics: 公開用のスナップショット → /metrics
```

| 配置 | 実装要素 |
| --- | --- |
| `cmd/switchbot-exporter/main.go` | コマンド振り分け、終了コード、シグナル、ビルド情報 |
| `internal/config/config.go` | フラグ、MAC 正規化、設定ファイルの検証 |
| `internal/ble/{bluez,advertisement}.go` | BlueZ 実装と広告の正規化 |
| `internal/protocol/{advertisement,decoder}.go` | 入力型、型付き機種データを保持する読取結果、共通のバイト演算 |
| `internal/models/registry.go` と `internal/models/<機種>/` | 種別と最小長の登録、各機種のデコードとメトリクス変換 |
| `internal/device/store.go` | MAC ごとの状態、Manufacturer Data の一時保持、スナップショット |
| `internal/metrics/` | Collector、共通系列、スキャン監視 |
| `internal/app/` | スキャンバックエンドの契約、常駐と単発スキャンの制御 |

`ble` と `device` は `protocol` の入力型を共有する。`protocol` は標準ライブラリだけに依存する。機種追加の手順は [開発ガイド](../guides/development.md#機種の追加) を参照する。`metrics` からは公開済みの状態だけを参照し、HTTP ハンドラーは Bluetooth の完了を待たない。スキャン処理は一つの goroutine が所有する。単発スキャンも同じ受信・デコード経路を使い、最後に JSON を一度だけ出力する。

BlueZ D-Bus には `github.com/godbus/dbus/v5`、Prometheus 出力には `github.com/prometheus/client_golang` を使う。デバイス状態はプロセスのメモリに保持し、再起動後は受信した広告から再構築する。

## 旧実装と異なる動作

| 項目 | Go 版の設計判断 |
| --- | --- |
| スキャン制御 | 起動直後の初回スキャン、開始間隔 10 秒、同時実行 1 件。復旧時は接続を作り直し、アダプターの電源操作は行わない |
| 到着順と入力エラー | Manufacturer Data の先行キャッシュに期限・件数上限を設ける。短い入力や型不明の入力から既存の正常値を壊さない |
| メトリクス公開 | 必要な値がそろった系列から公開する。部分データでも取得済みの系列を維持し、Python 版で起こり得るキー欠落エラーを回避する |
| CLI 出力 | MAC を小文字に統一して Python の出力に合わせる |
| 運用監視 | スキャン監視の 5 系列を追加する。移行時に Python ランタイム由来の系列の利用状況を別途確認する |

移植元の役割は `switchbot.py` が受信とデコード、`switchbot_scanner.py` が単発 JSON、`switchbot_exporter.py` が定期スキャンとメトリクス公開。Go 版では単一コマンドに統合した。
