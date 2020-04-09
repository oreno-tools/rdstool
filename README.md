# rdstool

## これは

* RDS (特に Aurora) を操作するコマンドラインツールです
* 現時点では Aurora のみで動作確認しています

## なぜ作ったのか

* 操作ミスを減らす (メンテナンスは手順書に沿って手を動かすだけにしたい)
* マネジメントコンソール前提の手順書が書き辛い

## 出来ること

* クラスタ内の DB インスタンス一覧取得
* DB インスタンスのインスタンスクラス変更
* DB インスタンスの再起動
* クラスタのフェイルオーバー
* 設定済パラメータグループのパラメータと値の確認
* 一部のパラメータについて値の変更

ちなみに, 以下は DB インスタンスのインスタンスクラスを変更している図です.

![](https://github.com/inokappa/rdstool/blob/master/docs/images/rdstool1.png?raw=true)

## 出来ないこと

* 全てのパラメータ値の変更
* DB インスタンスの作成, 起動, 停止

## こだわったところ

* 変更が発生するアクションでは, 必ず, yes or no で処理を継続を確認するようにした
* DB インスタンス名はタイポを防ぐ為, 矢印キーで選択出来るようにした

## はじめに

環境変数に DB クラスタ, DB パラメータグループ名を設定しておくと良いです.

```shell
export PARAMETER_NAME=oreno-db-parameter
export CLUSTER_NAME=x-kensho-cluster
```

AWS のリソースを操作するので, アクセスキー, シークレットアクセスキー又はプロファイル名, リージョン名を環境変数に指定しておきましょう.

```shell
export AWS_PROFILE=your-profile
export AWS_REGION=ap-northeast-1
export AWS_DEFAULT_REGION=ap-northeast-1
```

## インストール

```sh
# Get latest version
v=$(curl -s 'https://api.github.com/repos/inokappa/rdstool/releases' | jq -r '.[0].tag_name')
# For macOS
$ wget https://github.com/inokappa/rdstool/releases/download/${v}/rdstool_darwin_amd64 -O ~/bin/rdstool && chmod +x ~/bin/rdstool
# For Linux
$ wget https://github.com/inokappa/rdstool/releases/download/${v}/rdstool_linux_amd64 -O ~/bin/rdstool && chmod +x ~/bin/rdstool
```

## DB インスタンス一覧

クラスタに所属している DB インスタンスの一覧を取得したい場合, 以下のように実行します.

```shell
$ rdstool -instances
```

## DB インスタンスのインスタンスクラス変更

DB インスタンスのインスタンスクラス変更は以下のように実行します.

```shell
$ rdstool -modify -class=db.r5.large
```

## DB インスタンス再起動

DB インスタンスの再起動は以下のように実行します.

```shell
$ rdstool -restart
```

## クラスタのフェイルオーバー

クラスタのフェイルオーバーは以下のように実行します.

```shell
$ rdstool -failover
```

## 設定済みパラメータグループのパラメータと値の確認

DB インスタンスに設定されたパラメータグループのパラメータと値を客員するには, 以下のように実行します.

```shell
$ rdstool -param-name=${パラメータ名}
```

## shared_buffers パラメータの変更

一部のパラメータについて, rdstool を使って変更することが出来ます. 現時点で動作確認しているのは, `shared_buffers` のみです. 以下のように実行します.

```shell
$ rdstool -param-name=shared_buffers -ratio=0.5 -modify
```

上記の例は, パラメータ `shared_buffers` をインスタンスの搭載メモリ 50% に指定しています.
