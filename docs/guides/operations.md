# Linux 運用ガイド

初回は [クイックスタート](../../README.md) の手順でスキャンと `/metrics` の応答を確認してください。コマンドや `nametable.json` の詳細は [CLI と設定の仕様](../SPEC/cli.md) にあります。

## 広告を診断する

対象機器が通常の `scan` に現れない場合、`--raw` で BlueZ から届く広告を確認できます。例では 10 秒間観測し、標準出力をファイルに保存します。別のアダプターを使う場合は `hci0` を置き換えてください。

```sh
./switchbot-exporter scan --adapter hci0 --raw > advertisements.jsonl
```

`mac` と `fields` から対象機器の広告を探し、対象 UUID / Company ID を [ペイロード正規化仕様](../SPEC/ble.md#ペイロード正規化) と照合してください。ログを共有する前に端末識別情報を除いてください。実際の HCI パケットとの比較は [BLE PoC の検証手順](ble-poc.md#raspberry-pi-zero-w-での確認) を参照してください。

## systemd で稼働させる

対象アーキテクチャの Linux バイナリと、リポジトリ内の systemd Unit を用意します。リポジトリのルートで、`binary` を手元のバイナリのパスに置き換えて実行してください。既存の Python 版が動いている場合は、先に切替手順に従ってください。[systemd Unit](../../systemd/switchbot-exporter.service) · [切替と切り戻し](#切替と切り戻し)

```sh
binary=./dist/switchbot-exporter-linux-armv6
sudo install -d -m 0755 /usr/local/switchbot
sudo install -m 0755 "$binary" /usr/local/bin/switchbot-exporter
sudo install -m 0644 systemd/switchbot-exporter.service /etc/systemd/system/switchbot-exporter.service
```

新規配備では `sudo install -m 0600 ./nametable.json /usr/local/switchbot/nametable.json` で名前表を配置します。既存の名前表がある場合は、その内容を確認して引き継いでください。Unit は BlueZ の起動後に Exporter を root で実行し、`/usr/local/switchbot` を作業ディレクトリにします。

systemd に Unit を読み込ませ、自動起動を有効にしてサービスを起動します。

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now switchbot-exporter.service
systemctl status switchbot-exporter.service
sudo journalctl -u switchbot-exporter.service -n 30 --no-pager
```

プロセス異常終了後は systemd が 5 秒後に再起動する。SIGINT / SIGTERM ではスキャンをキャンセルして後始末を行い、その後 HTTP を最大 5 秒かけて終了する。Unit の停止上限は 15 秒。BLE の一時障害では HTTP を維持し、[スキャン監視](../SPEC/metrics.md#スキャン監視) に結果を反映する。[systemctl リファレンス](https://www.freedesktop.org/software/systemd/man/latest/systemctl.html)

### 起動後の確認

- `curl -fsS http://127.0.0.1:8000/metrics` で HTTP が応答すること。
- 1 回目のスキャンが終わった後に `switchbot_scan_success` が 1 になり、`switchbot_last_success_timestamp` が更新されること。
- 対象機器が近くにある場合は `switchbot_devices_seen` と機器別の `switchbot_update` を確認する。HTTP 応答があっても、スキャン失敗や未検出は起こり得る。

### よくある症状

| 症状 | 確認箇所 |
| --- | --- |
| 起動直後に終了する | ログに記載された設定パス、JSON の形式、MAC の重複、待受ポートの使用状況 |
| `switchbot_scan_success` が 0 のまま | BlueZ の稼働、`bluetoothctl list` / `bluetoothctl show` のアダプターと電源状態、D-Bus に接続できる実行ユーザー |
| スキャン成功で機器別の系列が出ない | `scan` で取得した MAC と名前表の一致、機器の距離、[公開対象の機種](../SPEC/metrics.md#系列とラベル) |
| 値は表示されるが鮮度が低い | `switchbot_update` と `switchbot_last_success_timestamp`。前者は最後にデコードできた測定、後者は最後に成功したスキャン |

## 切替と切り戻し

1. `systemctl show switchbot-exporter.service -p FragmentPath -p DropInPaths -p WorkingDirectory` で旧サービスの設定を確認する。旧 Unit・drop-in・Python スクリプト・名前表・置き換える既存バイナリを退避する。旧版の名前表は旧 Unit の作業ディレクトリも確認し、Go 版へ引き継ぐ。バックアップ先と旧 `/metrics` の取得時刻を記録する。
2. 同じアダプターを使う場合は旧サービスを停止した上で、Go 版を一時ポート（例 `127.0.0.1:18000`）で起動する。別アダプターを使える場合は並行比較できる。Prometheus 系列の厳密な比較には同じ入力を再生する [互換テスト](../SPEC/validation.md#検証項目) を使う。
3. 比較用の Go 版を停止し、並行稼働させていた旧サービスも停止する。Go 版と Unit を配備し、Unit を再読み込みしてサービスを起動する。`active`・`enabled`、ポート 8000 の応答、2 回以上のスキャン成功と機器別系列を確認する。旧版の `switchbot_update` が古い場合は、双方に存在する機器の系列名とラベルを比較し、照合できた台数を記録する。[systemd の配備手順](#systemd-で稼働させる)
4. `sudo systemctl restart switchbot-exporter.service` で再起動し、PID が変わり、最終成功時刻と機器別系列が再び更新されることを確認する。Grafana・PromQL の表示、Bluetooth 障害からの復旧、長時間稼働は [初期リリースの確認項目](../SPEC/validation.md#初期リリースの受入条件) に沿って記録する。
5. 起動できない、主要系列が欠ける、復旧しない場合は Go 版を停止し、退避した旧 Unit・drop-in・実行ファイルを元のパスへ戻す。名前表を変更していた場合は、退避した旧版の名前表も戻す。`sudo systemctl daemon-reload` と `sudo systemctl start switchbot-exporter.service` を実行し、旧 `/metrics` の更新を確認する。名前表や Prometheus の既存時系列に形式変換は不要。

Pi Zero W での切替と再起動の測定結果は [本番切替の記録](../records/verification.md#raspberry-pi-zero-w-への本番切替) を参照する。
