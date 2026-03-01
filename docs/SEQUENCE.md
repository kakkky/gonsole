# REPLシーケンス図

```mermaid
sequenceDiagram
    actor User
    participant Repl
    participant SymbolIndex
    participant Completer
    participant DeclRegistry
    participant Executor
    participant SessionSrcDir as 📁SessionSrcDir

    loop
        User->>Repl: 文字入力...
        Repl->>Completer: 条件に合致する候補はあるか？
        activate Completer
            Note over Completer: シンボルを探索
            Completer->>DeclRegistry: セレクタ式の場合、.の前は登録済みの変数か？
            alt Yes
                DeclRegistry-->>Completer:変数情報
                Completer->>SymbolIndex: 変数をレシーバとしたメソッドを探索
            else No
                Completer->>SymbolIndex: メソッド以外の要素を探索
            end
            Completer-->>Repl: 候補を返却
        deactivate Completer

        Repl-->>User: 候補を提示
        User->>Repl: 式/宣言文を決定(Enter)
        Repl->>Executor: 入力文字列を実行
        activate Executor
            Note over Executor: 入力情報を元に実行
            Executor->>Executor: 入力文字列をASTに解釈
            Executor->>Executor: メモリとして保持するASTキャッシュに追加
            Executor->>SessionSrcDir: 一時ディレクトリを生成
            Executor->>SessionSrcDir: 一時ディレクトリ内にGoファイルを作成
            Executor->>SessionSrcDir: 一時ファイルにASTのキャッシュを<br>ソースコードとしてフラッシュ
            Executor->>SessionSrcDir: `go.mod`を動的生成して配置<br>(pp/v3をrequire、プロジェクト本体をreplace)
            Executor->>SessionSrcDir: `go mod tidy`で`go.sum`を生成
            Executor->>SessionSrcDir: 一時ファイルに対して`go run`で実行
            alt 実行エラー
                Executor->>Executor: 今回の入力情報を<br>ASTキャッシュから削除
                Executor->>SessionSrcDir: 一時ファイルに更新されたASTのキャッシュを<br>ソースコードとしてフラッシュ
            end
            alt 出力がある
                Executor-->>User: 実行結果を標準出力で提示<br>(式の場合はpp.Printlnでフォーマット&シンタックスハイライト)
            end
            alt 式呼び出しの場合
                Executor->>Executor: ASTキャッシュから式呼び出しを削除
            else 宣言文の場合
                Executor->>SessionSrcDir: 一時ファイルをparseして型解決済みの宣言情報を取得
                Executor->>DeclRegistry: 宣言情報をレジストリに登録
            end
            Executor->>SessionSrcDir: 一時ディレクトリ配下の各ファイル削除後、一時ディレクトリを削除
            Executor-->>Repl:実行完了
        deactivate Executor
        Repl->>User: `>`を提示して入力待機
    end
```