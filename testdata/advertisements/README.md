# BLE 広告のテストデータ

元の Python 実装と Go 版の出力を同じ入力で比較するための合成フィクスチャです。

| ファイル | 内容と用途 |
| --- | --- |
| [python-synthetic.json](python-synthetic.json) | デコード分岐ごとの最小長の入力と、Python デコーダーの期待値。境界テストでは各分岐の末尾を 1 バイト短くした入力も使います |
| [python-metrics.json](python-metrics.json) | 同じ合成入力を Python Exporter に渡し、Gauge と Enum に設定されたサンプルを記録したものです |

どちらのファイルにも実機の MAC アドレスや測定値は含まれません。テストの観点は [検証・受入条件](../../docs/SPEC/validation.md#検証項目) を参照してください。
