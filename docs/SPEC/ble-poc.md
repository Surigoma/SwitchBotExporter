# BLE PoC の実行と検証

`cmd/switchbot-ble-poc` は BlueZ D-Bus の `Adapter1` で LE discovery を開始し、`Device1` の通知を 1 行 1 JSON オブジェクトで出力する。10 秒間、`hci0` を使うのが既定。標準エラーには終了時の受信件数またはエラーを出す。PoC は起動ごとに専用 D-Bus 接続を作り、終了時に自分の discovery セッションを解放する。

## ビルドとローカルの検証

```sh
go test ./...
go vet ./...
mkdir -p dist
GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build -trimpath \
  -o dist/switchbot-ble-poc-linux-armv6 ./cmd/switchbot-ble-poc
readelf -h -l -d dist/switchbot-ble-poc-linux-armv6
```

D-Bus の統合テストは Linux 上で隔離したバスと模擬 BlueZ を使用する。Debian / Ubuntu では `dbus-run-session`、Python 3、`python3-dbus`、`python3-gi` が必要。

```sh
go build -o dist/switchbot-ble-poc ./cmd/switchbot-ble-poc
dbus-run-session -- python3 internal/ble/testdata/bluez_mock.py dist/switchbot-ble-poc
```

統合テストは LE フィルター、開始と停止、Manufacturer Data の先行、同一 Service Data の反復、初期オブジェクト一覧に含まれる古い値の除外、Adapter 電源断・未検出、シグナルによる停止を検証する。

## Raspberry Pi Zero W での確認

ARMv6 バイナリを Pi に転送し、対象機器の MAC を指定して実行する。 `--mac` は省略すると全デバイスを観測する。

```sh
uname -m
bluetoothctl --version
./switchbot-ble-poc-linux-armv6 --adapter hci0 --duration 10s --mac 00:11:22:33:44:55 > bluez.jsonl
```

| キー | 意味 |
| --- | --- |
| `receivedAt` | 通知を受け取った UTC 時刻 |
| `source` | `interfacesAdded` または `propertiesChanged` |
| `mac` | 小文字の MAC アドレス |
| `rssi`、`rssiChanged` | 同じ窓で最後に観測した RSSI と、この通知に RSSI が含まれたか。未取得の場合は `rssi` を省略する |
| `serviceData` | UUID をキーにした BlueZ の値。値は元のバイト列を小文字の十六進文字列にしたもの |
| `manufacturerData` | Company ID の 4 桁十六進表記をキーにした BlueZ の値。値は同上 |
| `switchbotFields` | [候補識別子](ble.md#ペイロード正規化) に一致したデータ。 `adType` は数値、`id` は 4 桁の十六進表記、`payloadHex` は識別子を除いたバイト列 |

同じ時間帯に Pi の `btmon` で HCI 受信を観測し、MAC ごとに両方の AD Type、識別子、ペイロード、RSSI、同一データの反復を突き合わせる。スキャンを続けて実行し、既存の `Device1` からも通知されるか確認する。Adapter の電源断や BlueZ 再起動の確認は検証機で実施する。取得した JSON Lines と HCI ログを共有する場合は端末識別情報を除く。

## 判定状況

ARMv6 ターゲットのビルドでは ELF32 / ARM、動的インタープリターと動的セクションがないことを確認した。Go の単体テストと模擬 D-Bus の統合テストは通過した。

2026-11-28 に Raspberry Pi（`armv6l`、Raspbian 13.6、カーネル `6.18.60+rpt-rpi-v6`、BlueZ 5.82）で、既存の Python Exporter を稼働させたまま、一般ユーザーから 10 秒のスキャンを 2 回実行した。

| 観測項目 | 1 回目 | 2 回目 |
| --- | ---: | ---: |
| BlueZ の通知数 | 645 | 615 |
| SwitchBot の候補識別子を含む通知数 | 389 | 396 |
| 候補識別子を含む端末数 | 10 | 10 |
| うちローカルの旧名前表と一致する端末数 | 9 | 9 |
| Service Data が先だった端末数 | 2 | 9 |
| Manufacturer Data が先だった端末数 | 8 | 1 |

2 回とも同じ 10 台から `fd3d` の Service Data と `0969` の Manufacturer Data を確認した。同一ペイロードの反復通知と RSSI の更新も観測できた。2 回の終了後に既存 Exporter は稼働中で、開始時刻と再起動カウンターは変化しなかった。

HCI との同時採取では BlueZ 通知 636 件のうち SwitchBot のフィールド 615 件が、同一 MAC・AD Type・識別子・識別子を除くペイロードで HCI 側の広告に一致した。すべて 250 ms 以内に対応する広告があり、最も近い一致の時差は最大 78 ms。SwitchBot 通知の RSSI 更新 389 件も同一 MAC の HCI 値と一致した。BlueZ D-Bus を受信方式として採用する。HCI ログには端末識別情報が含まれるため、Git 管理外で照合した。

`0d00` と `0059` は合成入力で正規化を確認し、実機では未観測。BlueZ 再起動と Adapter OFF/ON は既存 Exporter に影響するため、実機での障害注入は保留する。一般ユーザーで BlueZ に接続でき、HCI の同時採取にだけ管理者権限が必要だった。
