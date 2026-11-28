# SwitchBot BLE Prometheus Exporter

SwitchBot の BLE Advertisement を BlueZ で受信し、`/metrics` を公開します。対応ターゲットは Raspberry Pi Zero W の Linux/ARMv6 です。既定では `:8000` で待ち受けます。

## ビルドと実行

Go 1.25 以上と [Task](https://taskfile.dev/docs/installation) を使います。

```sh
task test
task lint
task build
task build-armv6
```

公開する機器を `nametable.json` に登録します。キーが BLE MAC、値が Prometheus の `name` ラベルです。

```json
{"00:11:22:33:44:55": "example-meter"}
```

```sh
./dist/switchbot-exporter --config ./nametable.json
./dist/switchbot-exporter scan
./dist/switchbot-exporter scan --raw --scan-duration 10s
./dist/switchbot-exporter version
```

Exporter は登録済み機器を対象にします。単発スキャンは既定ですべての機器を観測します。単発スキャンにも `--config` を指定すると、登録済み機器に絞れます。

## 機種の追加・変更

機種固有のコードは [`internal/models/`](internal/models/) の機種別ディレクトリに置きます。たとえば [Meter TH](internal/models/meterth/) の `decode.go` はペイロードを読み、`metrics.go` はその値を公開系列に変換します。[種別表](internal/models/registry.go) で Service Data の種別 ID、必要バイト数、機種の実装を登録します。Curtain 3 の共通デコーダーは [`internal/models/curtain3/`](internal/models/curtain3/) にあります。

Pi での [BlueZ PoC](docs/SPEC/ble-poc.md)、[Prometheus の互換性仕様](docs/SPEC/metrics.md)、[systemd の配備・切り戻し手順](docs/SPEC/operations.md#切替と切り戻し) を参照してください。
