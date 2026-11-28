# Linux ビルドと GitHub リリース

## 配布形式

[Taskfile](../../Taskfile.yml) と [共通ビルドスクリプト](../../scripts/build-linux.sh) で Linux 版をビルドする。配布バイナリは CGO を無効にし、CPU と ELF のビット幅、Go のターゲット設定、共有ライブラリ依存の有無を検証する。実行時には BlueZ、system D-Bus、利用できる Bluetooth アダプターが必要。

| 配布対象 | ビルドコマンド | `dist/` 以下のバイナリ |
| --- | --- | --- |
| Linux x86-64 | `task build-linux ARCH=amd64` | `switchbot-exporter-linux-amd64` |
| Linux AArch64（64 ビット） | `task build-linux ARCH=arm64` | `switchbot-exporter-linux-arm64` |
| Linux ARMv7（32 ビット） | `task build-linux ARCH=arm ARM_VERSION=7` | `switchbot-exporter-linux-armv7` |
| Linux ARMv6（Pi Zero W など） | `task build-armv6` | `switchbot-exporter-linux-armv6` |

それ以外のアーキテクチャは `go tool dist list` で対象の `linux/<arch>` を調べ、`task build-linux ARCH=<arch>` でソースからビルドする。例えば `task build-linux ARCH=riscv64`。32 ビット ARM の `ARM_VERSION` の既定値は `7`。ビルド後の実行条件はディストリビューションの BlueZ と D-Bus 設定に依存する。

## CI と更新 PR

- [CI](../../.github/workflows/ci.yml) は push・Pull Request・手動実行で、テスト、race detector、lint、模擬 BlueZ、Linux の配布対象全てのクロスビルドを実行する。成功した実行の `switchbot-exporter-linux` アーティファクトからバイナリを取得できる。
- [Dependabot](../../.github/dependabot.yml) は Go モジュールと GitHub Actions を毎週確認する。更新 PR は通常の必須 CI を通す。
- `main` は PR と最新の基準ブランチに対する `Test and build` の成功が必要。force push とブランチ削除はルールで禁止されている。[保護ルール](https://github.com/Surigoma/SwitchBotExporter/rules/28284595)

## リリースを公開する

1. `main` に変更を取り込むと、[Release Please](../../.github/workflows/release.yml) が Conventional Commits から変更履歴と次の版をまとめる。Squash merge で取り込む場合は、対象の PR タイトルを `feat:`、`fix:` などの形式にする。
2. リリース PR に **Approve workflows to run** が表示されたら、書き込み権限のある利用者が実行を承認する。リリース PR の必須 CI が成功してからマージする。[GitHub のワークフローの起動条件](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/trigger-a-workflow#triggering-a-workflow-from-a-workflow)
3. **Release** ワークフローがタグとドラフト Release を作成する。タグをチェックアウトして配布対象それぞれをビルド・検証し、アーキテクチャごとの `.tar.gz`（バイナリと `LICENSE`）と共通の `SHA256SUMS` を添付する。
4. [Releases](https://github.com/Surigoma/SwitchBotExporter/releases) でタグ、変更履歴、アーカイブ 4 件、`SHA256SUMS` を確認して公開する。初回の公開条件は [検証チェックリスト](verification.md#初期リリースの確認項目) を参照する。[GitHub のリリース管理](https://docs.github.com/en/repositories/releasing-projects-on-github/managing-releases-in-a-repository)

初回バージョンは [Release Please 設定](../../release-please-config.json) の `0.1.0`。以降は [マニフェスト](../../.release-please-manifest.json) とマージ済みの Conventional Commits を使って次の版を算出する。Release Please はリポジトリの標準 `GITHUB_TOKEN` を使用する。

## 失敗したパッケージを作り直す

タグとドラフトは作成されたが添付に失敗した場合は、Actions の **Release** を `main` から手動実行して、対象タグを `release_tag` に指定する。`v0.1.0` のようなタグを指定すると、そのタグの内容で再ビルドし、検証後に同名の配布ファイルを既存ドラフトへ再添付する。空欄では選択したコミットをパッケージし、Actions アーティファクトに保存する。
