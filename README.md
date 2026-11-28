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

## GitHub CI と依存更新

[CI](.github/workflows/ci.yml) は push と Pull Request でテスト、lint、race detector、模擬 BlueZ の結合テストを実行します。ARMv6 バイナリは成功した Actions 実行の成果物から取得できます。実機での BLE 確認は [BlueZ PoC の手順](docs/SPEC/ble-poc.md) を参照してください。

[Dependabot](.github/dependabot.yml) は、設定が既定ブランチに入ると Go モジュールと GitHub Actions の更新を毎週確認し、Pull Request を作成します。

[Release Please](.github/workflows/release.yml) は `main` の Conventional Commits から変更履歴とバージョンをまとめたリリース PR を作成します。初回の予定バージョンは `v0.1.0` です。リリース PR をマージするとタグとドラフト Release が作成され、ARMv6 の `.tar.gz` と `SHA256SUMS` が添付されます。添付を確認してからドラフトを公開します。

Release Please は標準の `GITHUB_TOKEN` を使います。リリース PR に **Approve workflows to run** が表示されたら CI の実行を承認します。既存のドラフトに添付し直すときは、Actions の **Release** を `main` から手動実行し、既存のタグを `release_tag` に指定します。空欄ではアーカイブをビルドし、Actions の成果物に保存します。

## 機種の追加・変更

機種固有のコードは [`internal/models/`](internal/models/) の機種別ディレクトリに置きます。たとえば [Meter TH](internal/models/meterth/) の `decode.go` はペイロードを読み、`metrics.go` はその値を公開系列に変換します。[種別表](internal/models/registry.go) で Service Data の種別 ID、必要バイト数、機種の実装を登録します。Curtain 3 の共通デコーダーは [`internal/models/curtain3/`](internal/models/curtain3/) にあります。

Pi での [BlueZ PoC](docs/SPEC/ble-poc.md)、[Prometheus の互換性仕様](docs/SPEC/metrics.md)、[systemd の配備・切り戻し手順](docs/SPEC/operations.md#切替と切り戻し) を参照してください。
