# SwitchBot BLE Prometheus Exporter

## Go Rewrite — Implementation Specification

**Document Version:** 0.1  
**Status:** Draft / Implementation Ready  
**Language:** Go  
**Target Platform:** Raspberry Pi Zero W / ARMv6 / Raspbian Debian Trixie  
**Project Name:** switchbot-exporter（仮称）

---

# 1. プロジェクト概要

## 1.1 目的

既存のPython製SwitchBot BLE ExporterをGoで再実装する。

SwitchBotデバイスが発信するBLE Advertisementを直接受信・解析し、Prometheus形式でメトリクスを公開する。

SwitchBot Cloud APIやHub経由の通信には依存せず、BLEによる直接取得方式を維持する。

主な目的は以下。

- Pythonおよびbluepyへの依存を排除する
- Raspberry Pi Zero Wで安定して動作させる
- BLE Advertisementの解析処理を独立したモジュールに分離する
- Prometheusメトリクスの後方互換性を維持する
- Bluetooth障害発生時に自動復旧できるようにする
- 単一バイナリで配布・運用できるようにする

## 1.2 基本方針

以下を設計上の原則とする。

1. BLEからの直接取得を維持する。
2. 既存のSwitchBot独自プロトコル解析を移植する。
3. 既存のPrometheusメトリクスを変更しない。
4. BLE受信処理とプロトコル解析処理を完全に分離する。
5. ARMv6環境で動作することを最優先する。
6. エラー発生時にExporter全体を停止させない。
7. 外部APIやクラウドサービスに依存しない。

# 2. 動作環境

## 2.1 ターゲットハードウェア

| 項目 | 仕様 |
| --- | --- |
| Device | Raspberry Pi Zero W Rev 1.1 |
| CPU | ARMv6 |
| Architecture | armhf |
| RAM | 512MB |
| OS | Raspbian GNU/Linux 13 (Trixie) |
| Kernel | Linux |
| Bluetooth | 内蔵Bluetooth |
| Bluetooth Stack | BlueZ |

## 2.2 ビルド要件

以下の環境変数を指定してクロスコンパイルできること。

```bash
GOOS=linux
GOARCH=arm
GOARM=6
CGO_ENABLED=0
```

要件：

- CGOに依存しないこと
- 外部共有ライブラリに依存しないこと
- ARMv6向けバイナリを生成できること
- Goの標準ライブラリを積極的に利用すること

採用するGoバージョンはARMv6のサポートを確認したうえで決定する。

# 3. 現行システムの仕様

## 3.1 現行ファイル構成

| ファイル | 役割 |
| --- | --- |
| switchbot.py | BLE Advertisement受信・デバイス判定・デコード |
| switchbot_scanner.py | 単発スキャン・JSON出力 |
| switchbot_exporter.py | 定期スキャン・Prometheus Exporter |

Go版ではこれらの機能を統合する。

## 3.2 BLE Advertisementの解析仕様

現行実装では以下のAD Typeを解析対象としている。

| AD Type | 名称 |
| --- | --- |
| 0x16 | Service Data - 16-bit UUID |
| 0xFF | Manufacturer Specific Data |

対象とする識別子は以下。

| AD Type | 先頭4桁（現行16進文字列） |
| --- | --- |
| 0x16 | 3dfd |
| 0x16 | 000d |
| 0xFF | 6909 |
| 0xFF | 5900 |

先頭4桁は識別用として扱い、デコーダーにはその後のペイロードを渡す。

※ BlueZ D-BusではUUIDおよびCompany IDが異なる形式で公開される可能性があるため、PoCでバイトオーダーを確認し、現行実装と同じペイロードになることを検証する。

## 3.3 デバイス判定

Service Dataの先頭バイトからデバイスタイプを判定する。

```go
deviceType := payload[0] & 0x7f
```

現行実装のデバイスタイプ定義を維持する。

| ID | デバイス | デコーダー |
| --- | --- | --- |
| H | Bot | 対応 |
| B | Button Add Mode | 未対応 |
| b | Button | 対応 |
| L | Hub Add Mode | 未対応 |
| l | Hub | 未対応 |
| P | Hub Plus Add Mode | 未対応 |
| p | Hub Plus | 未対応 |
| F | Fan Add Mode | 未対応 |
| f | Fan | 未対応 |
| t | Meter TH Add Mode | 未対応 |
| T | Meter TH | 対応 |
| M | Hub Mini Add Mode | 未対応 |
| m | Hub Mini | 一部対応 |
| D | Contact Sensor Add Mode | 未対応 |
| d | Contact Sensor | 対応 |
| W | Outdoor Meter Add Mode | 未対応 |
| w | Outdoor Meter | 対応 |
| o | Smart Lock | 対応 |
| y | Keypad Touch | 対応 |
| C | Curtain Add Mode | 未対応 |
| c | Curtain | 対応 |
| [ | Curtain 3 Const Adv | 対応 |
| { | Curtain 3 Pair | 対応 |
| q | Ceiling Light | 対応 |
| s | Motion Sensor | 対応 |
| v | Hub 2 | 対応 |

Add Modeなど、現行実装でデコーダーが存在しないデバイスについては、認識機能を維持するが、メトリクス化は行わない。

## 3.4 Advertisementの到着順序

Service DataとManufacturer Dataの到着順序は保証されない。

現行実装ではManufacturer Dataが先に届いた場合、一時的にキャッシュし、Service Dataによるデバイス判定後にデコードする。

Go版でもこの動作を維持する。

必要な処理：

1. MACアドレス単位で状態を管理する。
2. Service Dataからデバイスタイプを判定する。
3. Manufacturer Dataが先行していた場合はキャッシュを参照する。
4. デバイスタイプ確定後にキャッシュをデコードする。
5. 処理済みキャッシュを破棄する。
6. 期限切れキャッシュを定期的に破棄する。

キャッシュは無制限に保持しない。

TTLは設定可能とし、初期値は30秒を提案する。実機でAdvertisementの到着間隔を確認して調整する。

# 4. 対応デバイスと取得情報

## 4.1 デバイス別デコード項目

### Bot

- enc
- mode
- state
- commit_flag
- group
- sync_utc
- battery

### Button

- enc
- battery

### Meter TH

- group
- battery
- alert.temp
- alert.hum
- temp
- hum

### Outdoor Meter

- battery
- sequence_num
- temp
- hum

### Contact Sensor

- enc
- scope_tested
- state.PIR
- state.HAL
- latest_trigger.PIR
- latest_trigger.HAL
- battery
- light_level
- act_counter.entrances
- act_counter.go_out
- act_counter.button_push

### Smart Lock

- battery
- sequence_num
- lock_info.calibrated
- lock_info.update_source
- lock_info.status.lock
- lock_info.status.door
- config.double_lock_mode
- config.unclosed_door_timeout_alarm
- config.pause_automatic_locking

### Keypad Touch

- battery

### Curtain

- allow_connection
- calibration
- battery
- movement
- position
- light_level
- device_chain

### Curtain 3

- direction
- calibration
- battery
- movement
- position
- light_level
- device_chain

### Ceiling Light

- sequence_num
- level
- status
- small_status
- color_temp
- channel

### Motion Sensor

- scope_test
- PIR
- battery
- latest
- LED_state
- IOT_state
- sensing_distance
- light_intensity

### Hub Mini

- latest_time

現在デコード処理が存在するものは移植対象とする。ただし、Hub Miniは現行Exporterではメトリクス化されていないため、初期リリースでは既存の公開メトリクスを増やさない。

### Hub 2

- sequence_num
- light_level
- temp
- hum

## 4.2 デコード上の注意

- 温度の符号処理を維持する。
- 小数点以下の温度情報を維持する。
- FahrenheitからCelsiusへの変換を維持する。
- バッテリー値のビットマスクを維持する。
- Enum相当の状態値を維持する。
- Smart Lockの状態テーブルを維持する。
- Contact Sensorのカウンターおよび時刻拡張処理を維持する。
- Curtain 3の識別子 `[` と `{` は同一のデコーダーを利用する。

不正なペイロードや長さ不足のデータはエラーとして記録し、Exporter全体を停止させない。

配列アクセス前に必ず必要なデータ長を確認すること。

# 5. システムアーキテクチャ

## 5.1 全体構成

```text
                  BlueZ
                    |
                    v
             BLE Adapter
                    |
                    v
             Advertisement
                    |
                    v
           Protocol Decoder
                    |
                    v
             Device Store
                    |
                    v
           Prometheus Metrics
                    |
                    v
             HTTP Server
                    |
                    v
              /metrics
```

各コンポーネントは独立させる。

## 5.2 BLE Backend

第一候補としてBlueZ D-Busを採用する。

利用ライブラリ候補：

- github.com/godbus/dbus/v5

BlueZのD-Bus APIを利用し、Advertisementの受信を行う。

必要な機能：

- Bluetooth Adapterの検出
- スキャン開始・停止
- RSSI取得
- Service Data取得
- Manufacturer Data取得
- デバイスMACアドレス取得
- Adapter状態監視
- D-Bus接続断からの復旧

BlueZ D-Busで現行実装と同等の情報が取得できるかを最初のPoCで確認する。

特に以下を重点検証する。

1. Service DataとManufacturer Dataの取得
2. UUIDおよびCompany IDの変換
3. 同一デバイスからの連続Advertisement通知
4. 両データの到着順序が異なるケース
5. RSSIの更新
6. Bluetooth Adapterの再接続

D-Bus経由で必要な情報を取得できない場合は、Raw HCI方式を代替案として検討する。

BLE Backendをインターフェース化し、将来の受信方式変更がDecoderに影響しない設計とする。

## 5.3 内部データ構造

BLE BackendからDecoderへ渡すデータは正規化する。

概念例：

```go
type Advertisement struct {
    MAC        string
    RSSI       int16
    ReceivedAt time.Time
    Fields     []ADField
}

type ADField struct {
    Type    uint8
    ID      uint16
    Payload []byte
}
```

`Payload` は識別子部分を除いたデータとする。

Service DataおよびManufacturer Dataの識別情報は保持する。

実際のBlueZからの変換処理はPoCで確定する。

## 5.4 Device Store

MACアドレス単位でデバイス状態を保持する。

```go
type DeviceState struct {
    MAC       string
    Type      byte
    TypeName  string
    RSSI      int16
    UpdatedAt time.Time
    Data      interface{}
}
```

実装時にはデバイス別の型安全な構造体を優先する。

必要な機能：

- デバイス状態の更新
- 最終受信時刻の保持
- RSSIの保持
- デバイスタイプ管理
- デコード済みデータの保持
- MACアドレスによる検索

データ取得が一時的に途絶えても、最後に取得した値は保持する。

現行Exporterと同様、未検出になっただけでメトリクスを削除しない。

# 6. Prometheus Exporter

## 6.1 基本仕様

HTTPポート：

```text
8000
```

エンドポイント：

```text
/metrics
```

デフォルトでは全インターフェースでListenする。

Prometheus公式Goクライアントを使用する。

```text
github.com/prometheus/client_golang
```

## 6.2 既存メトリクスの互換性

最重要要件。

既存のMetric名、ラベル名、ラベル値、Enum状態表現を変更しない。

### 共通メトリクス

| Metric | Labels |
| --- | --- |
| switchbot_batt | key, mac, name |
| switchbot_rssi | key, mac, name |
| switchbot_update | key, mac, name |

`switchbot_update` はUnix Timestampとする。

バッテリー情報が存在しない場合は、既存実装と同様にBattery Metricを生成・更新しない。

### Meter TH

- switchbot_meter_th_temp
- switchbot_meter_th_humid

### Outdoor Meter

- switchbot_meter_outdoor_temp
- switchbot_meter_outdoor_humid

### Contact Sensor

- switchbot_contact_sensor_state_PIR
- switchbot_contact_sensor_state_HAL
- switchbot_contact_sensor_light_level
- switchbot_contact_sensor_act_entrances
- switchbot_contact_sensor_act_go_out
- switchbot_contact_sensor_act_button_push

Enum状態：

PIR:

- No one moves
- Someone is moving

HAL:

- door close
- door open
- timeout not close
- undefined

Light:

- dark
- light

### Smart Lock

- switchbot_smartlock_sequence
- switchbot_smartlock_lock

Lock状態：

- Locked
- Unlocked
- Locking
- Unlocking
- LockingStop(Stuck)
- UnlockingStop(Stuck)
- Not fully locked(only for EU Lock Type)
- Undefined

### Curtain

- switchbot_curtain_position
- switchbot_curtain_light_level
- switchbot_curtain_movement

### Curtain 3

- switchbot_curtain3_position
- switchbot_curtain3_light_level
- switchbot_curtain3_movement

Movement状態：

- stationary
- movement

### Ceiling Light

- switchbot_ceiling_light_sequence
- switchbot_ceiling_light_level
- switchbot_ceiling_light_status
- switchbot_ceiling_light_small_status
- switchbot_ceiling_light_color_temp

### Motion Sensor

- switchbot_motion_sensor_latest
- switchbot_motion_sensor_led_status
- switchbot_motion_sensor_iot_status
- switchbot_motion_sensor_PIR
- switchbot_motion_sensor_light_intensity

PIR状態：

- No one moves
- Someone is moving

Light Intensity状態：

- Reserve
- dark
- bright

### Hub 2

- switchbot_hub_2_sequence
- switchbot_hub_2_light_level
- switchbot_hub_2_temp
- switchbot_hub_2_humid

## 6.3 Enumの実装

Python版ではPrometheusのEnumを使用している。

Go版ではGaugeVec等を利用し、同じ出力形式を再現する。

例えば以下の形式を維持する。

```text
switchbot_curtain3_movement{
  mac="...",
  name="kghr-curtain-left",
  switchbot_curtain3_movement="stationary"
} 1
```

単純な数値Gaugeに置き換えてはならない。

## 6.4 Exporter自身の監視

以下のメトリクスを追加する。

| Metric | Type | 用途 |
| --- | --- | --- |
| switchbot_scan_success | Gauge | 最終スキャン成功状態 |
| switchbot_scan_errors_total | Counter | スキャンエラー数 |
| switchbot_scan_duration_seconds | Histogram | スキャン所要時間 |
| switchbot_devices_seen | Gauge | 検出デバイス数 |
| switchbot_last_success_timestamp | Gauge | 最終成功時刻 |

これらは既存メトリクスとは独立して追加する。

# 7. スキャン制御

## 7.1 基本設定

| 項目 | デフォルト |
| --- | --- |
| Scan Duration | 5秒 |
| Scan Interval | 10秒 |
| Bluetooth Adapter | hci0 |
| HTTP Port | 8000 |

Scan Intervalはスキャン開始間隔の目標値とする。

同時に複数のスキャンを実行してはならない。

スキャンが予定時間を超えた場合は重複実行せず、次の実行機会に進む。

## 7.2 エラー処理

以下のエラーを個別に処理する。

- D-Bus接続断
- Bluetooth Adapter消失
- Adapter Powered=false
- Scan開始失敗
- Scan停止失敗
- Advertisement受信エラー
- 不正なペイロード
- Decoderエラー

エラー発生時もHTTP Serverは稼働を継続する。

D-Bus接続が切れた場合は再接続する。

再接続は指数バックオフを使用し、最大待機時間を設定する。

初期案：

- 最小待機：1秒
- 最大待機：60秒

Bluetooth Adapterの電源を無条件にOFF/ONする処理は実装しない。

現行のhciconfigによるリセット処理は廃止する。

## 7.3 Graceful Shutdown

SIGINT / SIGTERMを受け取った場合：

1. 新規スキャンを停止する。
2. 実行中のスキャンを終了させる。
3. BLEスキャンを停止する。
4. HTTP ServerをGraceful Shutdownする。
5. プロセスを終了する。

# 8. 設定ファイル

## 8.1 nametable.json

現行形式を維持する。

MACアドレスと表示名の対応表として使用する。

```json
{
  "00:11:22:33:44:55": "kghr-example"
}
```

上記は形式例であり、実際のMACアドレスではない。

MACアドレスは大文字・小文字を正規化して扱う。

Exporterでは登録済みMACアドレスをスキャン対象とする。

設定ファイルが存在しない、または対象デバイスが存在しない場合は、理由を明示して終了する。

## 8.2 設定項目

最低限以下を設定可能にする。

- config file path
- Bluetooth Adapter
- HTTP Listen Address
- Scan Duration
- Scan Interval
- Advertisement Cache TTL
- Log Level

デフォルト値を持ち、一般的な環境では設定ファイル以外の追加設定を必要としないこと。

# 9. CLI仕様

以下のコマンドを実装する。

```bash
# Exporter起動
switchbot-exporter

# 単発スキャン
switchbot-exporter scan

# バージョン
switchbot-exporter version
```

## 9.1 scan

現行switchbot_scanner.pyの代替機能。

デフォルトで10秒間スキャンし、デコード結果をJSONで標準出力する。

JSONの基本構造は現行形式を維持する。

```json
{
  "AA:BB:CC:DD:EE:FF": {
    "type": 84,
    "typeStr": "SwitchBot MeterTH (WoSensorTH)",
    "rssi": -52,
    "update": 1795364397,
    "data": {
      "battery": 88,
      "temp": 24.5,
      "hum": 50
    }
  }
}
```

上記は出力形式の説明用サンプル。

実際のデコード値は受信データに従う。

追加機能としてRaw Advertisementを出力できるデバッグモードを用意する。

## 9.2 version

以下を表示する。

- アプリケーションバージョン
- Git Commit
- Build Date
- Go Version

# 10. ソースコード構成

```text
switchbot-exporter/
├── cmd/
│   └── switchbot-exporter/
│       └── main.go
├── internal/
│   ├── ble/
│   │   ├── adapter.go
│   │   └── bluez.go
│   ├── protocol/
│   │   ├── advertisement.go
│   │   └── decoder.go
│   ├── models/
│   │   ├── registry.go
│   │   └── <model>/
│   │       ├── decode.go
│   │       └── metrics.go
│   ├── device/
│   │   └── store.go
│   ├── metrics/
│   │   └── exporter.go
│   ├── config/
│   │   └── config.go
│   └── app/
│       └── app.go
├── testdata/
│   └── advertisements/
├── go.mod
├── Taskfile.yml
├── README.md
├── LICENSE
└── .gitignore
```

責務を明確に分離する。

- ble: Bluetooth通信とAdvertisement取得
- protocol: 正規化したAdvertisementと共通のデコード処理
- models: 機種別のデコード、種別登録、メトリクス変換
- device: デバイス状態の管理
- metrics: Prometheusメトリクス
- config: 設定読み込み・検証
- app: 各コンポーネントの統合

# 11. テスト仕様

## 11.1 Decoder Unit Test

デバイスごとにテストを用意する。

- 正常データ
- データ長不足
- 不正なデータ
- 不明なデバイスタイプ
- 温度の正負
- Fahrenheit変換
- バッテリー値
- Enum値
- ビット演算

実機から取得したAdvertisementをテストフィクスチャとして保存する。

実データがないケースについて、架空のパケットを実機検証済みとして扱わない。

## 11.2 Advertisement Test

以下をテストする。

- Service Data先行
- Manufacturer Data先行
- 同一データの重複受信
- MACアドレスが異なる複数デバイス
- キャッシュのTTL切れ
- 未知のAD Type
- 未知の識別子

## 11.3 Prometheus Compatibility Test

Python版とGo版のメトリクス出力を比較する。

比較対象：

- Metric名
- Label名
- Label値
- Gauge値
- Enum状態
- Unix Timestamp
- デバイスごとのメトリクス数

既存のGrafanaダッシュボードおよびPromQLを変更せずに利用できること。

## 11.4 Integration Test

- BlueZとの接続
- スキャン開始・停止
- Advertisement取得
- HTTP Server起動
- /metrics応答
- D-Bus切断後の復旧
- Graceful Shutdown

## 11.5 ARMv6 Test

ARMv6向けにクロスコンパイルできること。

最終的にはRaspberry Pi Zero W実機で動作確認する。

# 12. ビルド・配布

Taskfileで以下を提供する。

```bash
task build
task test
task lint
task build-armv6
```

ARMv6ビルド成果物：

```text
dist/
└── switchbot-exporter-linux-armv6
```

バイナリ単体で動作すること。

実行時にGo、Python、pip、仮想環境などを必要としないこと。

# 13. systemd

既存のExporterを置き換えられるsystemd Unitを提供する。

想定パス：

```text
/usr/local/bin/switchbot-exporter
/usr/local/switchbot/nametable.json
/etc/systemd/system/switchbot-exporter.service
```

初期段階では現行環境の権限・Bluetoothアクセス条件を維持し、実機で確認してから最小権限化を検討する。

systemdのRestart設定を行い、プロセス異常終了時に再起動する。

設定例：

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

[Install]
WantedBy=multi-user.target
```

実際のBluetooth D-Busアクセス要件をPoCで確認すること。

# 14. 実装マイルストーン

## Milestone 1: BLE PoC

最優先。

BlueZ D-Busを利用し、Raspberry Pi Zero WでSwitchBotのAdvertisementを取得する。

確認事項：

- Service Data取得
- Manufacturer Data取得
- UUID / Company IDの変換
- MACアドレス取得
- RSSI取得
- 重複通知
- データ到着順序

この段階ではExporterを実装しない。

BlueZ D-Busで必要情報を取得できなければ、Raw HCI方式を再検討する。

## Milestone 2: Protocol Decoder

既存PythonのデコーダーをGoへ移植する。

各デバイスのテストを実装し、実機データで検証する。

## Milestone 3: Prometheus Exporter

既存メトリクスを再現する。

Python版との比較テストを行う。

## Milestone 4: 常駐化

スキャンループ、エラー処理、再接続、Graceful Shutdownを実装する。

## Milestone 5: 実機移行

Python版とGo版を比較しながら検証する。

問題がなければGo版へ切り替える。

Python版のバックアップは動作確認完了まで保持する。

# 15. 完成条件

以下をすべて満たした時点で初期リリースとする。

- [ ] Raspberry Pi Zero Wで動作する
- [ ] ARMv6向け単一バイナリを生成できる
- [ ] CGOに依存しない
- [ ] BlueZ経由でSwitchBot BLE Advertisementを取得できる
- [ ] 現行対応デバイスのデコードを維持する
- [ ] Service Data / Manufacturer Dataの到着順序に対応する
- [ ] 不正データでExporterが停止しない
- [ ] BLE障害後に自動復旧する
- [ ] 既存Prometheusメトリクスと互換性がある
- [ ] /metricsをHTTP公開する
- [ ] scanコマンドでJSONを取得できる
- [ ] systemdで常駐稼働できる
- [ ] Python版からの移行手順がREADMEに記載されている

# 16. Codexへの実装指示

この仕様書に従って実装すること。

ただし、最初から全機能を一括実装してはならない。

以下の順序を厳守する。

1. リポジトリ構成と実装計画を作成する。
2. BLE PoCを実装する。
3. Raspberry Pi Zero Wで実機検証する。
4. PoCの結果を報告する。
5. BLE受信方式を確定する。
6. Decoderを実装する。
7. Prometheus Exporterを実装する。
8. テストを実行する。
9. ARMv6向けビルドを確認する。
10. systemd対応と移行手順を実装する。

各マイルストーンでテスト結果と未解決事項を報告すること。

不明なプロトコル仕様を推測だけで実装しない。

既存Python実装の挙動と異なる場合は、意図的な変更であることを明記する。

特にPrometheusメトリクスの互換性を最優先とする。

パフォーマンス最適化は実測結果に基づいて行い、不要な複雑化を避ける。

**最初の成果物はBLE PoCとする。**
