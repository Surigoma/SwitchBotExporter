# Prometheus 出力

上位仕様の [§6](../../SPEC.md) と [Python Exporter](../../.orig/switchbot_exporter.py) に対応する。既定の `:8000/metrics` を `promhttp.HandlerFor` で公開し、独立した Registry にデバイス Collector とスキャン監視を登録する。デバイス系列の HELP は既存と同じ `Switch Bot`。

## 系列とラベル

| メトリクス | ラベル | 値 |
| --- | --- | --- |
| `switchbot_batt` | `key, mac, name` | `battery`。未取得時は系列を生成・更新しない |
| `switchbot_rssi` | `key, mac, name` | 取得済みの RSSI（dBm）。未取得時は系列を生成・更新しない |
| `switchbot_update` | `key, mac, name` | 取得済みの `UpdatedAt.Unix()`（秒）。サンプルの明示タイムスタンプは付けない |

`key` は次表の接頭辞そのもの。MAC は小文字のコロン区切り、`name` は設定ファイルの値。機種固有の Gauge は `mac, name`、Enum は `mac, name` とメトリクス名と同名の状態ラベルを持つ。接頭辞に以下の接尾辞を連結した文字列をメトリクス名とする。 `G` は Gauge、`E` は Enum 形式の Gauge 群。

| ID | 接頭辞（共通系列の `key`） | 接尾辞 → 値 |
| --- | --- | --- |
| `H` | `switchbot_bot` | 共通のみ |
| `b` | `switchbot_button` | 共通のみ |
| `T` | `switchbot_meter_th` | `_temp` → temp (G)、`_humid` → hum (G) |
| `d` | `switchbot_contact_sensor` | `_state_PIR` → state.PIR (E)、`_state_HAL` → state.HAL (E)、`_light_level` → light_level (E)、`_act_entrances` → act_counter.entrances (G)、`_act_go_out` → act_counter.go_out (G)、`_act_button_push` → act_counter.button_push (G) |
| `w` | `switchbot_meter_outdoor` | `_temp` → temp (G)、`_humid` → hum (G) |
| `o` | `switchbot_smartlock` | `_sequence` → sequence_num (G)、`_lock` → lock_info.status.lock (E) |
| `y` | `switchbot_keypad_touch` | 共通のみ |
| `c` | `switchbot_curtain` | `_position` → position (G)、`_light_level` → light_level (G)、`_movement` → movement (E) |
| `[`、`{` | `switchbot_curtain3` | `_position` → position (G)、`_light_level` → light_level (G)、`_movement` → movement (E) |
| `q` | `switchbot_ceiling_light` | `_sequence` → sequence_num (G)、`_level` → level (G)、`_status` → status (G)、`_small_status` → small_status (G)、`_color_temp` → color_temp (G) |
| `s` | `switchbot_motion_sensor` | `_latest` → latest (G)、`_led_status` → LED_state (G)、`_iot_status` → IOT_state (G)、`_PIR` → PIR (E)、`_light_intensity` → light_intensity (E) |
| `v` | `switchbot_hub_2` | `_sequence` → sequence_num (G)、`_light_level` → light_level (G)、`_temp` → temp (G)、`_humid` → hum (G) |

Hub Mini と未対応型は共通系列も出さない。真偽値は true を 1、false を 0 とする。既存デコーダーが取得していても表に載らないフィールドは CLI に出力する。

## Enum の状態表

| 対象 | 状態値（大文字・空白を保持） |
| --- | --- |
| Contact Sensor の PIR、Motion Sensor の PIR | `No one moves`、`Someone is moving` |
| Contact Sensor の HAL | `door close`、`door open`、`timeout not close`、`undefined` |
| Contact Sensor の Light | `dark`、`light` |
| Smart Lock の Lock（bit 値 0–7 の順） | `Locked`、`Unlocked`、`Locking`、`Unlocking`、`LockingStop(Stuck)`、`UnlockingStop(Stuck)`、`Not fully locked(only for EU Lock Type)`、`Undefined` |
| Curtain と Curtain 3 の Movement | `stationary`、`movement` |
| Motion Sensor の Light Intensity | `Reserve`、`dark`、`bright` |

取得済みの Enum ごとに定義内の全状態を並べ、現在値の系列を 1、他の系列を 0 にする。例えば Curtain 3 の停止時は以下の 2 本を出す。

```text
switchbot_curtain3_movement{mac="00:11:22:33:44:55",name="example",switchbot_curtain3_movement="movement"} 0
switchbot_curtain3_movement{mac="00:11:22:33:44:55",name="example",switchbot_curtain3_movement="stationary"} 1
```

初期実装は Prometheus テキスト形式を互換対象とする。Python クライアントも同形式では Enum の型を gauge として出力する。Python の OpenMetrics `stateset` 型を前提にしたスクレイプがあるかは切替前に確認する。Go 版は初期値では OpenMetrics を有効化しない。[Prometheus の出力形式](https://prometheus.io/docs/instrumenting/exposition_formats/)、[Python client の変換処理](https://github.com/prometheus/client_python/blob/master/prometheus_client/exposition.py)。

## 公開タイミング

- 窓の正常終了時に Device Store から取得済みの値を変換し、既存の公開系列に上書きする。更新キーはメトリクス名と全ラベルの組。既に公開した系列はデバイスが検出されなくなっても保持する。型変更で系列名や `key` が変わった場合もプロセス内では以前の系列を保持する。再起動時は測定を受理するまで対象系列が現れない。
- Collector は記述子を固定し、更新後の不変なスナップショットを一括で差し替える。各 scrape は 1 つのスナップショットを参照して `MustNewConstMetric` 相当で値を出力する。Enum の 0/1 が切り替わる途中の値が同一 scrape に混ざることを防ぐ。
- 元の Python クライアントが暗黙に公開していたプロセス／ランタイム系列は出力契約の確認対象に含める。初期設計の Registry は SwitchBot 系列のみ。既存のダッシュボードやアラートが `process_*` や `python_*` に依存していれば移行判定時に対応を決める。

## スキャン監視

| 系列 | 更新規則 |
| --- | --- |
| `switchbot_scan_success` | 初期 0。最終完了窓が成功なら 1、失敗なら 0 |
| `switchbot_scan_errors_total` | 失敗完了した試行につき 1 回加算。正常終了のためのキャンセルと単発の不正広告は除く |
| `switchbot_scan_duration_seconds` | 接続準備から後始末までの経過秒を成功・失敗ともに 1 回観測する。通常停止による中断は除く。初期バケットは 0.5、1、2.5、5、7.5、10、15、30 秒 |
| `switchbot_devices_seen` | 最終成功窓で認識した許可済み MAC のユニーク数。既知 S の受理または確定型の有効 M を条件とする。認識のみの型を含む。失敗時は最後の成功値を保持し、正常に 0 台なら 0 に更新する |
| `switchbot_last_success_timestamp` | 正常終了した直近の窓の完了時刻（Unix 秒）。成功前は 0 |

監視系列にデバイスラベルを付けない。障害対応時は `switchbot_scan_success` と `switchbot_last_success_timestamp` でスキャン全体を、`switchbot_update` で各デバイスのデータ鮮度を確認する。
