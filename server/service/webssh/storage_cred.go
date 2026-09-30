package webssh

// Credential SSH 认证凭据（可被多台主机复用）
type Credential struct {
	ID          string `yaml:"id" json:"id"`                                 // 凭据 ID（自动生成）
	Name        string `yaml:"name" json:"name"`                             // 凭据名称
	Description string `yaml:"description" json:"description"`               // 凭据描述
	User        string `yaml:"user" json:"user"`                             // 用户名
	AuthType    string `yaml:"authType,omitempty" json:"authType,omitempty"` // 认证类型："password" | "privateKey" | ""
	Password    string `yaml:"password,omitempty" json:"-"`                  // 密码（不序列化到 JSON）
	PrivateKey  string `yaml:"privateKey,omitempty" json:"-"`                // 私钥（不序列化到 JSON）
}

func (c *Credential) fields() (*string, *string, *string) { return &c.ID, &c.Password, &c.PrivateKey }

// credentialStore 凭据存储
type credentialStore = itemStore[Credential, *Credential]

// credentialPrepare 处理密码/私钥并据此计算认证类型
func credentialPrepare(item, old *Credential) {
	keepSecrets(item, old)
	switch {
	case item.PrivateKey != "":
		item.AuthType = "privateKey"
	case item.Password != "":
		item.AuthType = "password"
	default:
		item.AuthType = ""
	}
}
