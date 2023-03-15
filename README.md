# switch-traffic-to-mackerel

**これは趣味プロダクトです**

ネットワークスイッチなどに対してSNMPで問い合わせを行い、インターフェイスの通信量などの統計情報を取得するプログラムです。

レポジトリ名にある通り、[mackerel.io](https://ja.mackerel.io/)に情報を送ることを主な目的としています。

## 特徴

- GETBULKを用いて値を取得するので、比較的速く動作します。
- 取り込むインターフェイス名を正規表現で指定することができるので、取り込みたくないインターフェイスを除外できます。
- mackerelに対して、通信量をシステムメトリックとして投稿するため、このプログラムが異常終了した場合など送信が失敗している状態に、死活監視で気づくことができます。
- mackerelとの通信が途絶えた場合でもプログラム内部でキャッシュし、通信が再開できたときに一斉に送信します。

## 使い方

1. config.yaml.sample を config.yaml という名前でコピーします
2. config.yaml を開き加工します
3. `switch-traffic-to-mackerel -config config.yaml` で起動する

## 設定ファイルの内容

```yaml
community: public # (必須)取得する対象のスイッチなどの SNMP コミュニティ名を設定する
target: 192.2.0.1 # (必須)取得する対象のスイッチなどの IPアドレスを設定する
interface: # (オプション)取り込むインターフェイスをインターフェイス名を使って絞り込むことができます。includeとexcludeはそれぞれ排他です。
    include: "" # 取得時に取り込みたいインターフェイス名を正規表現で指定します
    exclude: "" # 取得時に取り込みたくないインターフェイス名を正規表現で指定します
mibs: # (オプション)取り込みたい情報を設定できます。無指定時は、以下に示されるMIBについての情報が取り込まれます
    - ifHCInOctets
    - ifHCOutOctets
    - ifInDiscards
    - ifOutDiscards
    - ifInErrors
    - ifOutErrors
# 機器によっては ifHCInOctets、ifHCOutOctets への対応ができない場合があります。その場合は、以下を明示的に指定する必要があります
#   - ifInOctets
#   - ifOutOctets
debug: false # (オプション) true時、デバッグ表示を有効にします。取り込むインターフェイス名およびその値を表示します
dry-run: false # (オプション) true時、mackerel への送信を抑制します。mackerel についての情報が設定ファイルに含まれてない場合は、強制的に true となります。
skip-linkdown: false # (オプション) downしているインターフェイスについては取り込みをスキップするオプションです
mackerel: # (オプション)Mackerel に送信する時のパラメータ
    name: "" # (オプション)Mackerel に登録するホスト名
    x-api-key: xxxxx # (必須) Mackerel の APIキー
    host-id: xxxxx # (オプション) Mackerel でのホストID、無指定時の場合、プログラム内で自動的に取得し、設定ファイルを更新します。
```

## v0.0.1 からの移行

v0.0.1 までは設定ファイルを基本的に使用していませんでした。そのため設定ファイルを作成する必要があります。

以下のようなコマンドラインで起動させていた場合は、後述するようなYAMLになります

```
export MACKEREL_API_KEY=xxxx
./switch-traffic-to-mackerel -target 192.0.2.1 -mibs ifHCInOctets,ifHCOutOctets -include-interface 'ge-0/0/\d+$|ae0$' -skip-down-link-state -name sw1
```

```yaml
community: public
target: 192.0.2.1
mibs:
  - ifHCInOctets
  - ifHCOutOctets
interface:
  include: ge-0/0/\d+$|ae0$
skip-linkdown: true
mackerel:
    host-id: <192.0.2.1.id.txt というような ${target}.id.txt というファイルにホストIDが記録されているので転記してください>
    x-api-key: xxxx # 環境変数を読まなくなりましたので、直接記述してください
    name: sw1
```


