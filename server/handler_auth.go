package server

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"
)

// 登录
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	if UserMap[loginReq.Username] == loginReq.Password {
		token := md5sum(loginReq.Username + time.Now().String())
		Sessions[token] = time.Now().Add(24 * time.Hour)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"token": token,
			"user":  loginReq.Username,
		})
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
}

// 退出
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.URL.Query().Get("token")
	}

	delete(Sessions, token)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Logged out"})
}

// md5
func md5sum(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
