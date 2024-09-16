package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// OAuth2の設定
var (
	oauthConfig *oauth2.Config
	state       = "randomstate" // 認証リクエストの検証用ランダム文字列
)

func init() {
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("クレデンシャルファイルの読み込みに失敗しました: %v", err)
	}

	oauthConfig, err = google.ConfigFromJSON(b, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("OAuth2設定の作成に失敗しました: %v", err)
	}

	oauthConfig.RedirectURL = "http://localhost:8080/callback"
}

func main() {
	// 認証開始エンドポイント
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	})

	// 認証後にリダイレクトされるコールバックエンドポイント
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("state") != state {
			log.Printf("Stateが一致しません")
			http.Error(w, "Stateが一致しません", http.StatusBadRequest)
			return
		}

		code := r.FormValue("code")
		token, err := oauthConfig.Exchange(context.Background(), code)
		if err != nil {
			log.Printf("トークン交換に失敗しました: %v", err)
			http.Error(w, "トークン交換に失敗しました", http.StatusInternalServerError)
			return
		}

		// トークンを使ってGmail APIにアクセス
		client := oauthConfig.Client(context.Background(), token)
		srv, err := gmail.NewService(context.Background(), option.WithHTTPClient(client))
		if err != nil {
			log.Fatalf("Gmailサービスの作成に失敗しました: %v", err)
		}

		// メッセージリストを取得
		user := "0x706f6b6f@gmail.com"
		resp, err := srv.Users.Messages.List(user).MaxResults(10).Do()
		if err != nil {
			log.Fatalf("メッセージリスト取得に失敗しました: %v", err)
		}

		for _, v := range resp.Messages {

			message, err := srv.Users.Messages.Get(user, v.Id).Do()
			if err != nil {
				log.Fatalf("メッセージ取得に失敗しました: %v", err)
			}
			fmt.Fprintf(w, "id: %s\n", message.Id)
			fmt.Fprintf(w, "short message: %s\n", message.Snippet)
			fmt.Fprintf(w, "raw message: %v\n", message.Raw)
		}
		// for _, v := range resp.Messages {
		// 	fmt.Fprintf(w, "id: %s \n", v.Id)
		// 	fmt.Fprintf(w, "status: %d \n", v.HTTPStatusCode)
		// 	fmt.Fprintf(w, "thread id: %s \n", v.ThreadId)
		// }
	})

	// サーバーを起動
	fmt.Println("サーバーを起動しています: http://localhost:8080/login")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
