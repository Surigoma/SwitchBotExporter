# SwitchBot BLE Prometheus Exporter Go版詳細設計

対象は Raspberry Pi Zero W Rev 1.1、Linux/ARMv6、Raspbian Trixie。BLE Advertisement をローカルの Bluetooth アダプターから取得し、SwitchBot のデバイス情報を JSON と Prometheus 形式に変換する。[上位仕様](../../SPEC.md) を要件とし、[Python デコーダー](../../.orig/switchbot.py)、[単発スキャナー](../../.orig/switchbot_scanner.py)、[Exporter](../../.orig/switchbot_exporter.py)、[systemd Unit](../../.orig/switchbot_exporter.service) を互換性の根拠とする。

| 文書 | 設計範囲 |
| --- | --- |
| [BLE 受信とスキャン制御](ble.md) | BlueZ PoC、D-Bus、正規化、スキャン周期、復旧 |
| [Decoder と Device Store](protocol.md) | バイト配置、デバイス種別、結合、保持 |
| [Prometheus 出力](metrics.md) | 系列名、ラベル、Enum、監視メトリクス |
| [CLI・検証・移行](operations.md) | CLI と設定、配布、受入条件、切り戻し |

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
| `internal/ble/{adapter,bluez}.go` | スキャンのインターフェース、BlueZ 実装、正規化 |
| `internal/protocol/{advertisement,decoder}.go` | 入力型、型付き機種データを保持する読取結果、共通のバイト演算 |
| `internal/models/registry.go` と `internal/models/<機種>/` | 種別と最小長の登録、各機種のデコードとメトリクス変換 |
| `internal/device/store.go` | MAC ごとの状態、Manufacturer Data の一時保持、スナップショット |
| `internal/metrics/exporter.go` | 既存の系列定義、Collector、スキャン監視 |
| `internal/app/app.go` | Exporter と単発スキャンの処理、受信から公開までの接続 |

`ble` と `device` は `protocol` の入力型を共有する。 `protocol` は標準ライブラリだけに依存する。機種の追加では `internal/models/<機種>/` にデコーダーを置き、固有の系列があれば同じディレクトリに変換を実装し、種別表で登録する。 `metrics` からは公開済みの状態だけを参照し、HTTP ハンドラーは Bluetooth の完了を待たない。スキャン処理は一つの goroutine が所有する。単発スキャンも同じ受信・デコード経路を使い、最後に JSON を一度だけ出力する。

外部 Go ライブラリは D-Bus の `github.com/godbus/dbus/v5` と Prometheus の `github.com/prometheus/client_golang` を候補とする。BlueZ 固有の処理を受信側に閉じ込めると、PoC で Raw HCI へ変更する場合も、デコード・系列・CLI の契約を維持できる。デバイス状態はプロセスメモリで保持し、再起動後は Advertisement から再構築する。

## 旧実装と異なる動作

| 項目 | Go 版の設計判断 |
| --- | --- |
| スキャン制御 | 起動直後の初回スキャン、開始間隔 10 秒、同時実行 1 件。復旧時は接続を作り直し、アダプターの電源操作は行わない |
| 到着順と入力エラー | Manufacturer Data の先行キャッシュに期限・件数上限を設ける。短い入力や型不明の入力から既存の正常値を壊さない |
| メトリクス公開 | 必要な値がそろった系列から公開する。部分データでも取得済みの系列を維持し、Python 版で起こり得るキー欠落エラーを回避する |
| CLI 出力 | MAC を小文字に統一して Python 出力と一致させる。上位仕様の JSON にある大文字 MAC は形式例として扱う |
| 運用監視 | スキャン監視の 5 系列を追加する。移行時に Python ランタイム由来の系列の利用状況を別途確認する |

BlueZ D-Bus を採用し、確認できた識別子・ペイロードと残る実機確認は [BLE PoC の判定](ble-poc.md#判定状況) に記す。キャッシュ上限、D-Bus タイムアウト、部分データの扱いなど、上位仕様で未定義の値は各文書に初期設計として記す。
