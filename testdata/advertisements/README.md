# Advertisement テストデータ

`python-synthetic.json` は各デコード分岐の最小長に合わせた合成入力と、元の Python デコーダーによる期待値です。各分岐の末尾 1 バイトを除いた入力も境界テストに使用します。`python-metrics.json` は同じ合成データを元の Python Exporter に渡し、Gauge と Enum に設定したサンプルを記録したものです。どちらにも実機の MAC アドレスや測定値は含まれません。
