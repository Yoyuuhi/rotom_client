# rotom
API 開発中にクライアントを立てたくない、またはクライアント実装待ちの段階で API のレスポンスを確認するために使用するツールです。
```
cp sample.env .env // 適宜に修正
cp sample.request.yml request.yml // 適宜に修正
go run main.go
```
定数などの修正ポイント：
- ローカルホストポート
- (ログインが必要であれば) main.go#init() 認証情報まわり

現時点動作確認したメソッド：
GET
POST
PATCH
