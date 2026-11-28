# SwitchBot Exporter のドキュメント

初めて使う場合は [クイックスタート](../README.md) から始めてください。

## 手順

| やりたいこと | 参照先 |
| --- | --- |
| systemd で稼働させる、障害を調べる、Python 版から切り替える | [Linux 運用ガイド](guides/operations.md) |
| ローカルでテストする、機種を追加する、CPU 負荷を調べる | [開発ガイド](guides/development.md) |
| 模擬 BlueZ でテストする、実機の BLE 広告を HCI と照合する | [BLE PoC の検証手順](guides/ble-poc.md) |
| Linux 向けにビルドする、GitHub でリリースを公開する | [ビルドとリリース](guides/release.md) |

## 仕様

[上位仕様](../SPEC.md) から CLI、BLE 受信、機種別デコード、Prometheus 出力、検証・受入条件を確認できます。コードを追うときは [詳細仕様と実装構成](SPEC/README.md) を参照してください。

## 検証記録

| 確認したいこと | 参照先 |
| --- | --- |
| BlueZ 採用の根拠、実機で観測できた識別子と RSSI | [BLE PoC の結果](records/ble-poc.md) |
| Python 版との互換性、Pi Zero W での本番切替、未確認項目 | [互換性と本番切替の検証記録](records/verification.md) |
| 連続スキャンを見送った理由、採用した負荷対策と測定値 | [常駐受信方式の比較](records/stream.md) |

バージョンごとの変更は [変更履歴](../CHANGELOG.md) を参照してください。
