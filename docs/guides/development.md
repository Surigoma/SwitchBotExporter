# 開発ガイド

リポジトリのルートで実行します。ビルドには [go.mod](../../go.mod) が指定する Go と Task が必要です。コードを追うときは [パッケージ構成](../SPEC/README.md#実行経路とパッケージ) を参照してください。

## 開発時のコマンド

```sh
task test
task lint
go test -race ./...
task build
task build-linux ARCH=amd64
```

`task test` は Go のテストを実行し、`task lint` は `gofmt` による整形の差分と `go vet` の結果を確認します。race detector は対応する開発ホストで実行してください。Linux の配布対象とアーカイブ形式は [リリース作業](release.md) を参照する。

## CPU 使用率を調べる

CPU の内訳を実機で調べるときは `-tags pprof` でビルドし、環境変数 `SWITCHBOT_PPROF=127.0.0.1:6060` を与えて起動する。`net/http/pprof` がその番地で応答するので、`curl -o cpu.pb.gz 'http://127.0.0.1:6060/debug/pprof/profile?seconds=120'` で採取し、手元の `go tool pprof -top -cum cpu.pb.gz` で読む。タグなしの配布ビルドには含まれない。基準実機での測定例は [ストリーム化の記録](../records/stream.md#負荷の内訳) にある。

## 機種の追加

1. `internal/models/<機種>/` にデコーダーを置き、必要な場合は固有系列の変換も同じディレクトリに置く。[Meter TH の実装](../../internal/models/meterth/) が例になる。
2. [種別表](../../internal/models/registry.go) に機種 ID、名前、必要なペイロード長、デコーダー、公開メトリクスの接頭辞を登録する。
3. [デコード仕様](../SPEC/protocol.md) と [メトリクス表](../SPEC/metrics.md) を更新し、最小長と境界値、互換性をテストする。

## 変更を検証する

変更箇所に対応する [検証項目](../SPEC/validation.md#検証項目) を確認してください。広告を使う互換テストの入力と期待値は [合成フィクスチャ](../../testdata/advertisements/README.md)、BlueZ の模擬テストや Raspberry Pi での採取方法は [BLE PoC の検証手順](ble-poc.md) にあります。
