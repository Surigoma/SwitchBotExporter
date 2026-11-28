# BLE 受信とスキャン制御

BlueZ D-Bus から取得した通知を正規化し、単発スキャンと常駐の両方で使う。受信方式を選定した根拠は [BLE PoC の結果](../records/ble-poc.md) を参照する。

## スキャンバックエンドの契約

```go
type ScanFunc func(context.Context, string, string, time.Duration, func(ble.Observation) error) error

type Advertisement struct {
    MAC        string
    RSSI       *int16
    ReceivedAt time.Time
    Fields     []ADField
}

type ADField struct {
    Type    uint8
    ID      uint16
    Payload []byte
}
```

`ScanFunc` の文字列引数は Adapter 名と任意の対象 MAC。`ble.Observation` は正規化した `protocol.Advertisement` と、`scan --raw` で使う UUID / Company ID 別のペイロードを持つ。コールバックはスキャン内で直列に呼び、ペイロードの独立したコピーを渡す。`RSSI=nil` は未取得。`ReceivedAt` は通知の受信時刻で、BLE パケットの送信時刻を表さない。BlueZ の一回の通知から複数フィールドが得られることもあれば、別々の通知になることもある。`Fields` が空のイベントは RSSI の更新に使える。標準出力の書き込みエラーがコールバックから返った場合はスキャンを終了し、単発 CLI は終了コード 1 を返す。

## BlueZ D-Bus

1. 専用の system bus 接続を作る。接続の終了と `NameOwnerChanged` による `org.bluez` の所有者変更を監視する。BlueZ の通知は現在の所有者と対象オブジェクトを照合する。
2. `ObjectManager.InterfacesAdded`、`InterfacesRemoved`、`Properties.PropertiesChanged` を購読する。`PropertiesChanged` の購読範囲は下表のとおり。`GetManagedObjects` で指定の `Adapter1` と `Powered=true` を確認し、配下の `Device1` の `Address` と `Adapter` から対応表を作る。初期取得の広告や RSSI は受信データとして使わない。
3. 対象 Adapter の `SetDiscoveryFilter` に `Transport="le"`、`DuplicateData=true` を設定し、`StartDiscovery` を呼ぶ。購読は開始要求前に済ませる。開始要求の送信時から応答までの通知は一時保存し、成功応答後に受信順で処理する。開始失敗時は破棄する。BlueZ の discovery セッションは共有されるので、受け取った通知はアプリ側でも Adapter と対象 MAC で絞る。
4. スキャン中の `InterfacesAdded` に含まれる `Device1` の値と、`PropertiesChanged` で実際に通知された項目を正規化する。更新通知に含まれない AD フィールドは補完しない。未登録のパスは `Address` と `Adapter` の取得で補完し、その応答を新規ペイロードには使わない。削除または無効化されたプロパティは対応表・RSSI キャッシュから消す。
5. 受信期間の終了後は通知の取り込みを止め、自分のセッションを `StopDiscovery` で解放して専用接続を閉じる。開始や停止の応答が不明でも接続を閉じることで自分のセッションを解放し、次の試行では購読とオブジェクト探索からやり直す。D-Bus 接続の準備、呼び出しと後始末にはそれぞれ期限を設ける（初期上限は各 3 秒）。後始末はキャンセル済みのスキャンとは別の期限付き context で実行する。窓の終端を受信時刻で判定し、その時刻以降の通知を次の窓に持ち越さない。

### `PropertiesChanged` の購読範囲

| 条件 | 購読範囲 |
| --- | --- |
| 名前表が 1〜64 件 | アダプターのパスと、登録 MAC に対応するデバイスパス（例 `/org/bluez/hci0/dev_00_11_22_33_44_55`）。範囲外の機器からの `PropertiesChanged` は D-Bus で配送されない |
| 名前表なし、または 65 件以上 | アダプターとその配下のデバイス |

### 通知キューと失敗条件

通知の受け口は上限 256 件の順序付きキュー。上限に達した後に通知が来ると受信エラーとしてスキャンを失敗させる。スキャン中にアダプターの電源が切れる、アダプターが削除される、BlueZ の所有者が変わる、または接続が切れる場合も、そのスキャンを失敗として扱う。

インターフェースの詳細は各 API 資料を参照する。[BlueZ Adapter1](https://github.com/bluez/bluez/blob/master/doc/org.bluez.Adapter.rst)、[BlueZ Device1](https://github.com/bluez/bluez/blob/master/doc/org.bluez.Device.rst)、[D-Bus ObjectManager と Properties](https://dbus.freedesktop.org/doc/dbus-specification.html)、[godbus/dbus/v5](https://pkg.go.dev/github.com/godbus/dbus/v5)。

## ペイロード正規化

| AD Type | Python が受け付ける先頭バイト列 | 数値 ID | BlueZ の辞書キー |
| --- | --- | --- | --- |
| `0x16` | `3d fd` | `0xfd3d` | `0000fd3d-0000-1000-8000-00805f9b34fb` |
| `0x16` | `00 0d` | `0x0d00` | `00000d00-0000-1000-8000-00805f9b34fb` |
| `0xff` | `69 09` | `0x0969` | Company ID `0x0969` |
| `0xff` | `59 00` | `0x0059` | Company ID `0x0059` |

Raw AD の識別子 2 バイトを little endian で読んだ値を `ID` とする。BlueZ の `ServiceData` は UUID 文字列、`ManufacturerData` は uint16 のキーを持つ。辞書の値をそのまま `Payload` として渡す。Manufacturer Data 側の先頭 6 バイトもペイロードに残す。HCI と照合した識別子は `fd3d` と `0969`。`0d00` と `0059` は合成入力で確認した。[BLE PoC の結果](../records/ble-poc.md#実機での観測結果)

同じ通知内では Service → Manufacturer の順で処理し、同一 AD Type に対象 ID が複数あれば上表の順を使う。通知間の到着順は保つ。

## 実行周期と復旧

- Exporter は HTTP の待受確立後に最初のスキャンを開始する。既定の開始間隔は 10 秒、受信時間は開始成功から 5 秒。開始枠は初回スキャンを起点とし、スキャン中やバックオフ中に過ぎた枠は飛ばす。開始・停止を含む試行が長引いても並列に実行しない。
- 接続断、Adapter 消失・電源 OFF、開始・停止失敗、受信キューあふれ等は窓を失敗させる。次の試行はスキャン間隔の制約に加え、失敗完了から 1、2、4、8、16、32、60 秒（以降 60 秒）の待機を満たす時刻に行う。連続失敗回数は窓が正常に完了した時点でリセットする。
- 不正な辞書値やペイロードは対象項目を破棄する。不正な辞書は debug ログに記録し、デコードの警告はスキャン内で 1 秒に 1 回まで記録する。検出 0 件でも開始・受信・停止が正常なら成功とする。
- D-Bus 断の後は接続・購読・パス対応表を再構築する。再接続中も HTTP は応答する。アダプターが OFF のままなら再試行を続ける。停止要求では新規スキャンとバックオフ待機を取り消し、実行中のスキャン後始末を待ってから HTTP を終了する。
