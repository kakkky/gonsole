# gonsole

Goプロジェクトの関数やメソッドを、REPL形式で対話的に実行できるCLIツールです。
Ruby on Railsの`rails console`のように、Goコードの関数・変数・構造体・メソッドを即座に試せます。

## 特徴

- Goプロジェクト内の、変数・定数・関数を参照可能
- コンソール内で変数や定数を定義し、それを用いて対話的に関数/メソッドの実行が可能
- 充実した補完機能で、呼び出す関数等のパッケージの選択から式の記述までがスムーズ

## インストール

```sh
go install github.com/kakkky/gonsole/cmd/gonsole@latest
```

または、リポジトリをcloneしてビルド:

```sh
git clone https://github.com/kakkky/gonsole.git
cd gonsole/cmd/gonsole
go build -o gonsole
```

## 使い方（クイックスタート）
この[サンプルプロジェクト](https://github.com/kakkky/gonsole-example)を用いて使い方を説明します。

### 起動
プロジェクトルートで以下を実行:
```sh
gonsole
```
すると、以下のような画面が出てきます。
```sh
  ____   ___   _   _  ____    ___   _      _____
 / ___| / _ \ | \ | |/ ___|  / _ \ | |    | ____|
| |  _ | | | ||  \| |\___ \ | | | || |    |  _|
| |_| || |_| || |\  | ___) || |_| || |___ | |___
 \____| \___/ |_| \_||____/  \___/ |_____||_____|


 Interactive Golang Execution Console

> 
```

`go mod init`を実行するなど、プロジェクトを初期化していないと起動時にエラーが出ます。

`>`の記号が出ていればgonsoleの起動成功です。この記号に続いてコードを記述し、実行する準備が整ったことを意味します。

また、この時`tmp/gonsolexxxxxxx/main.go`という一時ファイルが生成されます。このファイルはコード実行のために重要なので編集しないようにします。
```
├── tmp
│   └── gonsole784534083
│       └── main.go
```
このファイルは、コンソールの終了（`Ctrl + C`）と共に自動的に削除されます。

### Goコードの実行

#### パッケージの選択
入力に合わせてパッケージの候補が出ます。Tabキーを押して選択しましょう。
今回は`animal`パッケージの要素を呼び出すことにします。

![alt text](<スクリーンショット 2025-08-13 1.24.59.png>)
![alt text](image.png)


#### 変数定義
メソッド呼び出しや関数の引数に入れるために変数を定義することができます。
以下の関数を呼び出して、`dog`という変数に格納してみましょう。
```go
// 犬のコンストラクタ
func NewDog(name string, age int) *Dog {
	return &Dog{
		BaseAnimal: BaseAnimal{
			Name:  name,
			Age:   age,
			Fed:   false,
			Tired: false,
		},
		Breed: DefaultBreed,
	}
}
```

![alt text](image-5.png)


これで定義できました。

![alt text](image-7.png)


`var`による宣言でもOKです。
また、構造体リテラルを選択した場合は、フィールドが自動補完されます。

![alt text](image-6.png)

定義した変数を以下のように評価して確認することもできます。

![alt text](image-8.png)


#### メソッド呼び出し
上で定義した変数`dog`をレシーバとしてメソッドを呼び出してみます。
以下のようにメソッドの候補を選択します。

![alt text](image-9.png)
![alt text](image-10.png)

また、いちいち変数に格納しなくてもメソッドチェーンでも呼び出せます。

![alt text](image-11.png)
![alt text](image-12.png)
![alt text](image-13.png)

### 同名のパッケージ名が存在した場合（importパス選択モード）


### エラー検知


## 充実した補完機能


## ⚠️現状対応できていないケース