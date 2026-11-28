# Decoder と Device Store

上位仕様の [§3.2–4、§5.3–5.4](../../SPEC.md) と [Python デコーダー](../../.orig/switchbot.py) に対応する。

## 種別表と入力長

`p` は識別子を除いたペイロード。以下の添字は 0 始まり、スライス範囲の右端は含まない。Service Data（S）の `p[0] & 0x7f` が ASCII 種別 ID になる。 `—` は値のデコード対象外を示す。すべての S は種別判定だけでも 1 バイト必要。デコード前にフィールド全体の最小長を検証し、超過バイトは読み飛ばす。

| ID | `typeStr` | S 最小長 | Manufacturer Data（M）最小長 |
| --- | --- | ---: | ---: |
| `H` | SwitchBot Bot (WoHand) | 3 | — |
| `B` | WoButton [Add Mode] | — | — |
| `b` | WoButton | 3 | — |
| `L` | SwitchBot Hub (WoLink) [Add Mode] | — | — |
| `l` | SwitchBot Hub (WoLink) | — | — |
| `P` | SwitchBot Hub Plus (WoLink Plus) [Add Mode] | — | — |
| `p` | SwitchBot Hub Plus (WoLink Plus) | — | — |
| `F` | SwitchBot Fan (WoFan) [Add Mode] | — | — |
| `f` | SwitchBot Fan (WoFan) | — | — |
| `t` | SwitchBot MeterTH (WoSensorTH) [Add Mode] | — | — |
| `T` | SwitchBot MeterTH (WoSensorTH) | 6 | — |
| `M` | SwitchBot Mini (HubMini) [Add Mode] | — | — |
| `m` | SwitchBot Mini (HubMini) | — | 13 |
| `D` | SwitchBot Contact Sensor [Add Mode] | — | — |
| `d` | SwitchBot Contact Sensor | 9 | — |
| `W` | SwitchBot Outdoor Meter [Add Mode] | — | — |
| `w` | SwitchBot Outdoor Meter | 3 | 11 |
| `o` | SwitchBot Smart Lock | 3 | 9 |
| `y` | SwitchBot Keypad touch | 3 | — |
| `C` | SwitchBot Curtain [Add Mode] | — | — |
| `c` | SwitchBot Curtain | 5 | — |
| `[` | SwitchBot Curtain 3 [Const Adv] | 5 | — |
| `{` | SwitchBot Curtain 3 [Pair] | 5 | — |
| `q` | SwitchBot Ceiling light | — | 11 |
| `s` | SwitchBot Motion Sensor | 6 | — |
| `v` | SwitchBot Hub 2 | — | 16 |

`m` は Python が長さ 7 以上で `p[8:13]` の存在分を読み、データがなければ 0 を返す。Go 版は 5 バイトがそろった場合だけ値を採用する。実機で長さ 13 未満の正規バリアントが得られた場合は、形式ごとの境界を確認して暫定値を見直す。

## バイト配置

`b(i,n)` は `p[i]` の bit n（LSB が 0）、`be(a:b)` は `p[a:b]` の big endian 符号なし整数。 `0/1` で並べた文字列は bit が 0、1 のときの順。配列は 0 起点。バッテリーを持つ種別では `battery = p[2] & 0x7f`。温度を持つ種別の共通式は `bit7 == 1` なら正、0 なら負とし、絶対値を `float64(整数バイト & 0x7f) + float64(小数バイト & 0x0f)/10` とする。湿度バイトの bit 7 が 1 のときは `(temp - 32) / 1.8` で Celsius に変換し、`hum = 湿度バイト & 0x7f` とする。

| デバイスと入力 | フィールドと抽出式 |
| --- | --- |
| Bot / Button S | `enc` は `2*b(0,7)+b(1,4)`。順に `no encryption`、`encryption algorithm 1 (the same as the Bot)`、`encryption algorithm 2`、`encryption algorithm 3`。両方に `battery` |
| Bot S 追加 | `mode` は `b(1,7)` の `one state mode` / `on/off state mode`。 `state` は `b(1,6)` の `on` / `off`。 `commit_flag` は `b(1,4)` の `service data no data update` / `service data has data update`。 `group` は `p[1]` の bit 0–3 に対応する `A,B,C,D` のうち有効なものを順に並べる。 `sync_utc` は `b(2,7)` の `no need to synchronize` / `the Device has passed a time synchronization cycle (10 days by default) and requires a time synchronization` |
| Meter TH S | `group` は Bot と同じ。 `battery`。 `alert.temp` は `(p[3]>>6)&3` で `no alert`、`low temp alert`、`high temp alert`、`temp alert`。 `alert.hum` は `(p[3]>>4)&3` で `no alert`、`low humidity alert`、`high humidity alert`、`humidity alert`。温度の小数・整数・湿度バイトは `p[3], p[4], p[5]` |
| Outdoor Meter S / M | S は `battery`。M は `sequence_num=p[6]`、温度の小数・整数・湿度バイトは `p[8], p[9], p[10]` |
| Contact Sensor S | `enc` は `b(0,7)` の `No encryption` / `With encryption, encryption method 1`。 `scope_tested` と `state.PIR` はともに `b(1,7)`。前者は `Not tested` / `Tested`、後者は `No one moves` / `Someone is moving`。 `state.HAL` は `(p[3]>>1)&3` で `door close`、`door open`、`timeout not close`、`undefined`。 `latest_trigger.PIR=be(4:6)+65536*b(3,7)`、`latest_trigger.HAL=be(6:8)+65536*b(3,6)`、各 `0x1ffff` は `-1`。 `battery`。 `light_level` は `b(3,0)` の `dark` / `light`。 `act_counter.entrances=p[8]>>6`、`act_counter.go_out=(p[8]>>3)&3`、`act_counter.button_push=p[8]&7` |
| Smart Lock S / M | S は `battery`。M は `sequence_num=p[6]`、`lock_info.calibrated` は `b(7,7)` の真偽値、`lock_info.update_source` は `b(7,3)` の `primary_lock` / `secondary_lock`、`lock_info.status.lock` は `(p[7]>>4)&7` で[状態表](metrics.md#enum-の状態表)。 `lock_info.status.door` は `b(7,2)` の `closed` / `opened`。 `config.double_lock_mode` と `config.unclosed_door_timeout_alarm` は `b(8,7)` と `b(8,6)` の真偽値、`config.pause_automatic_locking` は `b(8,1)` の `inactived` / `activated` |
| Keypad Touch S | `battery` |
| Curtain / Curtain 3 S | Curtain の `allow_connection` は `b(1,7)` の真偽値。Curtain 3 の `direction` は同じ bit の `left` / `right`。両方とも `calibration` は `b(1,6)` の `not calibrated` / `calibrated`。 `battery`。 `movement` は `b(3,7)` の `stationary` / `movement`。 `position=p[3]&0x7f`、`light_level=p[4]>>4`、`device_chain=p[4]&0x0f` |
| Ceiling Light M | `sequence_num=p[6]`、`level=p[7]&0x7f`、`color_temp=be(8:10)`、`status` と `small_status` は `b(10,7)` と `b(10,6)` の真偽値、`channel=1+b(10,0)` |
| Motion Sensor S | `scope_test` は `b(1,7)` の `Not tested` / `Tested`、`PIR` は **`b(1,6)`** の `No one moves` / `Someone is moving`。 `battery`。 `latest=be(3:5)+65536*b(5,7)`。 `LED_state` と `IOT_state` は `b(5,5)` と `b(5,4)` の真偽値。 `sensing_distance` は `(p[5]>>2)&3` の `Long`、`Middle`、`Short`、`Reserve`。 `light_intensity` は `p[5]&3` の `Reserve`、`dark`、`bright`、`Reserve` |
| Hub Mini M | `latest_time=be(8:13)` を `uint64` に保持 |
| Hub 2 M | `sequence_num=p[11]`、`light_level=p[12]&0x0f`、温度の小数・整数・湿度バイトは `p[13], p[14], p[15]` |

値は機種ごとの型付き struct とし、M と S が別々に値を持つ場合は取得済みの部分を区別できる構造にする。未取得は nil または明示的な有無で表し、0 や false は有効な値とする。JSON には取得済みの値を出す。値を持つ `group` の空リストは `[]`。Decoder は結果を返してから Store に反映するので、途中で入力不備が見つかっても部分的な書き込みは残らない。シーケンス番号は値として保存し、重複排除や時刻判定には用いない。

## Device Store

- Exporter は正規化 MAC を `nametable.json` で照合してから処理する。単発 `scan` は既定で登録不要。対象の AD Type / ID がある通知で MAC の状態を作る。型未確定の JSON は `type=0`、`typeStr="Undefined"`、`data={}` とする。未取得 RSSI と更新時刻は、CLI でそれぞれ `-999`、`-1` として出力する。
- S が既知の型を示し、適用される長さ検証を通過したら型を確定する。既知だがデコーダーがない型も識別する。初回の未知 ID は数値と `Undefined` を CLI に残せる。確定済みの型を未知 ID や長さ不足では変更しない。既知の別機種に変わった場合は古いデコード済みの機種データをクリアする。Curtain 3 の `[` と `{` は共通機種として扱う。
- M 先行時は MAC ごとに受信順で保留する。有効な S で型が確定したら未期限切れの M を受信順に適用し、処理済みのエントリーを消す。同じ通知内では S と保留 M を処理してから、その通知に含まれる新しい M を適用する。未対応型の確定時はキャッシュを解放する。M の長さは型が判明した時点で検証する。
- M キャッシュの TTL 初期値は 30 秒。 `now >= receivedAt + TTL` で期限切れとし、広告処理時および 1 秒ごとのアプリタイマーで破棄する。初期上限は 1 MAC あたり 8 件、全体で 256 件、1 ペイロード 255 バイト。追加時に上限を超えれば古い順に破棄し、件数上限による破棄は抑制付きで記録する。タイマーと受信イベント間の状態更新は Store 内で直列化する。Decoder 済みデータに TTL は適用しない。
- 対応デコーダーの対象項目を受理した時点で `UpdatedAt` を元の受信時刻で進める。キャッシュ再生時は遡らせない。Hub 2 などの種別判定だけの S、RSSI 単独、解釈対象外の M では測定の更新時刻を進めない。Python はデコーダーの no-op 分岐でも時刻を更新していたため、この点は意図的に変更する。
- RSSI は対象 MAC に対するスキャン中の最新の実測値を保持する。対象 AD が到着する前の RSSI が同じ窓で取得できた場合、Backend から引き継げる。Python は主に S 処理時に RSSI を更新していたため、Go 版では M や RSSI 単独通知でも値が進む場合がある。取得済み値は欠落通知で消さない。
- スキャン中の Store は永続する作業状態。窓が正常に終了したら値のコピーを公開する。失敗した窓では公開状態を維持する。失敗までに正常に受理した値は記録した元時刻のまま次の成功時に公開できる。スキャン未検出や接続断で記録済みの値を削除しない。
