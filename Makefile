default: ## ヘルプを表示する
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

test: ## test テストの実行
	@go test -v

build: ## バイナリをビルドする
	@./build.sh main.go

release: ## バイナリをリリースする. 引数に `_VER=バージョン番号` を指定する.
	@ghr -u oreno-tools -r rdstool v${_VER} ./pkg/
