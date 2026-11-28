# BLE 受信とスキャン制御

上位仕様の [§3.2–3.4、§5.2–5.3、§7、§14–16](../../SPEC.md) に対応する。

現在の PoC のビルド・検証手順は [BLE PoC の実行と検証](ble-poc.md) を参照する。

## Backend 契約

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
2. `ObjectManager.InterfacesAdded`、`InterfacesRemoved`、`Properties.PropertiesChanged` を購読する。 `GetManagedObjects` で指定名の `Adapter1` と `Powered=true` を確認し、対象 Adapter 配下の `Device1` から `Address` と `Adapter` の対応表を初期化する。初期取得の Service Data、Manufacturer Data、RSSI は受信データとして使わない。
3. 対象 Adapter の `SetDiscoveryFilter` に `Transport="le"`、`DuplicateData=true` を設定し、`StartDiscovery` を呼ぶ。購読は開始要求前に済ませる。開始要求の送信時から応答までの通知は一時保存し、成功応答後に受信順で処理する。開始失敗時は破棄する。BlueZ の discovery セッションは共有されるので、受け取った通知はアプリ側でも Adapter と対象 MAC で絞る。
4. スキャン中の `InterfacesAdded` に含まれる `Device1` の値と、`PropertiesChanged` で実際に通知された項目を正規化する。更新通知に含まれない AD フィールドは補完しない。未登録のパスは `Address` と `Adapter` の取得で補完し、その応答を新規ペイロードには使わない。削除または無効化されたプロパティは対応表・RSSI キャッシュから消す。
5. 受信期間の終了後は通知の取り込みを止め、自分のセッションを `StopDiscovery` で解放して専用接続を閉じる。開始や停止の応答が不明でも接続を閉じることで自分のセッションを解放し、次の試行では購読とオブジェクト探索からやり直す。D-Bus 接続の準備、呼び出しと後始末にはそれぞれ期限を設ける（初期上限は各 3 秒）。後始末はキャンセル済みのスキャンとは別の期限付き context で実行する。窓の終端を受信時刻で判定し、その時刻以降の通知を次の窓に持ち越さない。

通知の受け口は順序付きの有限キューとする。PoC の初期上限は 256 通知で、満杯時は黙って捨てず受信エラーとして窓を失敗させる。通知処理によるブロックで復旧や終了が遅れないことを PoC で測定する。Adapter の Powered 変更、削除、BlueZ の所有者変更、接続断がスキャン中に発生した場合は、その窓を失敗させる。BlueZ の通知が物理パケットの一対一表現になるか、辞書が過去の値を含むかは実機で検証する。

API 資料 [BlueZ Adapter1](https://github.com/bluez/bluez/blob/master/doc/org.bluez.Adapter.rst)、[BlueZ Device1](https://github.com/bluez/bluez/blob/master/doc/org.bluez.Device.rst)、[D-Bus ObjectManager と Properties](https://dbus.freedesktop.org/doc/dbus-specification.html)、[godbus/dbus/v5](https://pkg.go.dev/github.com/godbus/dbus/v5)。

## ペイロード正規化

| AD Type | Python が受け付ける先頭バイト列 | 数値 ID 候補 | BlueZ の辞書キー候補 |
| --- | --- | --- | --- |
| `0x16` | `3d fd` | `0xfd3d` | `0000fd3d-0000-1000-8000-00805f9b34fb` |
| `0x16` | `00 0d` | `0x0d00` | `00000d00-0000-1000-8000-00805f9b34fb` |
| `0xff` | `69 09` | `0x0969` | Company ID `0x0969` |
| `0xff` | `59 00` | `0x0059` | Company ID `0x0059` |

Raw AD の識別子 2 バイトを little endian で読んだ値を `ID` とする案。BlueZ の `ServiceData` は UUID 文字列、`ManufacturerData` は uint16 のキーを持つ。識別子で種別を絞り、辞書の値をそのまま `Payload` として渡す案を採る。PoC で BlueZ の値と HCI の AD 部分を比較し、識別子が取り除かれていること、Manufacturer 側の先頭 6 バイトを含むペイロードが Python の `value[4:]` と一致することを確認する。同じ通知の辞書内では無順序なため、Service → Manufacturer の順で処理する。同一 AD Type に対象 ID が複数あれば上表の順を使う。別の通知の順序は保つ。

## 実行周期・失敗時の動作

- Exporter は HTTP の待受確立後に最初のスキャンを開始する。通常の開始スロットは初回を起点に 10 秒間隔で進め、スキャン中およびバックオフ中に過ぎたスロットは飛ばす。受信窓は開始成功時から 5 秒。開始・停止を含む試行が長引いても並列に実行しない。
- 接続断、Adapter 消失・電源 OFF、開始・停止失敗、受信キューあふれ等は窓を失敗させる。次の試行はスキャン間隔の制約に加え、失敗完了から 1、2、4、8、16、32、60 秒（以降 60 秒）の待機を満たす時刻に行う。連続失敗回数は窓が正常に完了した時点でリセットする。
- 不正な辞書値やペイロードは対象項目だけを破棄し、理由を構造化ログに記録する。短い広告が続いてもログを抑制できるよう、理由単位で同種の出力を制限する。スキャンの継続が可能なら成功可否は変えない。Zero 件でもスキャン API の開始・受信窓・終了が正常なら成功とする。
- D-Bus 断の後は接続・購読・パス対応表を再構築する。再接続中も HTTP は応答する。アダプターが OFF のままなら再試行を続ける。停止要求では新規スキャンとバックオフ待機を取り消し、実行中のスキャン後始末を待ってから HTTP を終了する。

## PoC の合格条件

Pi 実機で BlueZ・カーネルの版、アダプター、権限、各採取条件を記録する。BlueZ イベントと同時刻帯の HCI 観測を照合し、MAC、RSSI、両 AD Type、識別子、値の境界、同一データの反復、独立した到着と順序、複数スキャン窓での再受信を評価する。RSSI だけの通知や既存 Device1 に対する通知も確認し、古い辞書値で測定時刻を更新する問題がないか確かめる。Bluetooth サービス再起動、Adapter OFF → ON と再接続中の停止を確認する。捕捉した実機データと合成パケットは由来を明示する。

必要な機種のデータや更新を BlueZ から安定して取得できない場合は、欠けた情報と再現条件を記録して Raw HCI の検討に戻る。それまでは BlueZ での互換性を未検証とする。
