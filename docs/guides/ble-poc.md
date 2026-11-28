# BLE PoC の検証手順

`cmd/switchbot-ble-poc` は BlueZ D-Bus の `Adapter1` で LE discovery を開始し、`Device1` の通知を 1 行 1 JSON オブジェクトで出力する。既定は `hci0` で 10 秒間。標準エラーには終了時の受信件数またはエラーを出す。PoC は起動ごとに専用 D-Bus 接続を作り、終了時に自分の discovery セッションを解放する。

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
dbus-run-session -- /usr/bin/python3 internal/ble/testdata/bluez_mock.py dist/switchbot-ble-poc
```

統合テストは LE フィルター、開始と停止、Manufacturer Data の先行、同一 Service Data の反復、初期オブジェクト一覧に含まれる古い値の除外、Adapter 電源断・未検出、シグナルによる停止を検証する。

## Raspberry Pi Zero W での確認

ARMv6 バイナリを Pi に転送し、例の MAC を対象機器の値に置き換えて実行します。`--mac` を省略すると全デバイスを観測します。

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
| `switchbotFields` | 対象識別子に一致したデータ。`adType` は数値、`id` は 4 桁の十六進表記、`payloadHex` は識別子を除いたバイト列の十六進表記。[対象識別子](../SPEC/ble.md#ペイロード正規化) |

同じ時間帯に Pi の `btmon` で HCI 受信を観測し、MAC ごとに両方の AD Type、識別子、ペイロード、RSSI、同一データの反復を突き合わせる。スキャンを続けて実行し、既存の `Device1` からも通知されるか確認する。Adapter の電源断や BlueZ 再起動の確認は検証機で実施する。取得した JSON Lines と HCI ログを共有する場合は端末識別情報を除く。

評価条件は [BLE 受信方式の評価条件](../SPEC/validation.md#ble-受信方式の評価条件)、実機での照合結果は [BLE PoC の結果](../records/ble-poc.md) を参照してください。
