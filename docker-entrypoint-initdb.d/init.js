// 下記、一般ユーザー情報の保存先を指定（admin DB）
db = db.getSiblingDB('admin');

// ======================================================
// DBの一般ユーザーを登録（アプリ・GUI → DBへの接続時に利用）
// ======================================================
db.createUser({
    // 認証情報
    user:           "user",
    pwd:            "password",
    mechanisms:     ["SCRAM-SHA-256"], 

    roles: [
        // admin DB と照合する権限
        {
            role:   "read",
            db:     "admin"
        },
        // image_uploader_db に対する読み書き権限
        {
            role:   "readWrite",
            db:     "image_uploader_db"
        }
    ]
});

print("DBの一般ユーザー登録完了 \n以降、アプリ・GUIからDBへの接続が可能となります");