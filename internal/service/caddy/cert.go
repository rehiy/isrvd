package caddy

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	pkgcaddy "isrvd/pkgs/caddy"
)

// ─── TLS 证书 CRUD ───

// CertForm TLS 证书统一编辑模型
//
// 通过 Source 区分三种来源：
//   - file：load_files，Certificate/Key 是文件路径
//   - pem：load_pem，Certificate/Key 是 PEM 文本
//   - automate：automation.policies[].subjects 中的 host
type CertForm struct {
	Key         string   `json:"key,omitempty"`         // 复合主键 <source>-<index>，仅响应使用
	Source      string   `json:"source"`                // file / pem / automate
	Certificate string   `json:"certificate,omitempty"` // file: 路径；pem: PEM 文本
	KeyContent  string   `json:"keyContent,omitempty"`  // 私钥（路径或 PEM 文本）
	Tags        []string `json:"tags,omitempty"`
	Format      string   `json:"format,omitempty"`  // 仅 file 类型使用
	Subject     string   `json:"subject,omitempty"` // 仅 automate 使用：host 名称
}

// CertList 列出所有证书（合并三种来源）
func (s *Service) CertList(ctx context.Context) ([]CertForm, error) {
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CertForm, 0)
	if cfg.Apps == nil || cfg.Apps.TLS == nil {
		return out, nil
	}
	tls := cfg.Apps.TLS
	if tls.Certificates != nil {
		for i, f := range tls.Certificates.LoadFiles {
			out = append(out, CertForm{
				Key:         buildCertKey(CertSourceFile, i),
				Source:      CertSourceFile,
				Certificate: f.Certificate,
				// KeyContent 不返回：file 来源是路径引用，路径本身是安全信息
				Tags:   f.Tags,
				Format: f.Format,
			})
		}
		for i, p := range tls.Certificates.LoadPEM {
			out = append(out, CertForm{
				Key:         buildCertKey(CertSourcePEM, i),
				Source:      CertSourcePEM,
				Certificate: p.Certificate,
				// KeyContent 不返回：私钥不在列表接口暴露；编辑时客户端留空=保留原值
				Tags: p.Tags,
			})
		}
	}
	if tls.Automation != nil {
		idx := 0
		for _, policy := range tls.Automation.Policies {
			for _, subject := range policy.Subjects {
				out = append(out, CertForm{
					Key:     buildCertKey(CertSourceAutomate, idx),
					Source:  CertSourceAutomate,
					Subject: subject,
				})
				idx++
			}
		}
	}
	return out, nil
}

// CertCreate 创建证书
func (s *Service) CertCreate(ctx context.Context, req CertForm) error {
	if err := validateCertForm(req, true); err != nil {
		return err
	}
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return err
	}
	tls := ensureTLS(cfg)

	switch req.Source {
	case CertSourceFile:
		ensureCerts(tls).LoadFiles = append(tls.Certificates.LoadFiles, pkgcaddy.TLSLoadFile{
			Certificate: req.Certificate,
			Key:         req.KeyContent,
			Tags:        req.Tags,
			Format:      req.Format,
		})
	case CertSourcePEM:
		ensureCerts(tls).LoadPEM = append(tls.Certificates.LoadPEM, pkgcaddy.TLSLoadPEM{
			Certificate: req.Certificate,
			Key:         req.KeyContent,
			Tags:        req.Tags,
		})
	case CertSourceAutomate:
		appendAutomateSubject(tls, req.Subject)
	}

	return s.client.ConfigLoad(ctx, cfg)
}

// CertUpdate 更新证书（按 key 定位）
func (s *Service) CertUpdate(ctx context.Context, key string, req CertForm) error {
	source, index, err := parseCertKey(key)
	if err != nil {
		return err
	}
	if req.Source != "" && req.Source != source {
		return fmt.Errorf("不支持跨来源更新（%s → %s）", source, req.Source)
	}
	req.Source = source
	if err := validateCertForm(req, false); err != nil {
		return err
	}

	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return err
	}
	tls := ensureTLS(cfg)

	switch source {
	case CertSourceFile:
		if tls.Certificates == nil || index < 0 || index >= len(tls.Certificates.LoadFiles) {
			return fmt.Errorf("证书不存在")
		}
		tls.Certificates.LoadFiles[index] = pkgcaddy.TLSLoadFile{
			Certificate: req.Certificate,
			// 私钥留空则保留原值（客户端编辑时可不回填路径）
			Key:    pickSecretStr(req.KeyContent, tls.Certificates.LoadFiles[index].Key),
			Tags:   req.Tags,
			Format: req.Format,
		}
	case CertSourcePEM:
		if tls.Certificates == nil || index < 0 || index >= len(tls.Certificates.LoadPEM) {
			return fmt.Errorf("证书不存在")
		}
		tls.Certificates.LoadPEM[index] = pkgcaddy.TLSLoadPEM{
			Certificate: req.Certificate,
			// 私钥留空则保留原值
			Key:  pickSecretStr(req.KeyContent, tls.Certificates.LoadPEM[index].Key),
			Tags: req.Tags,
		}
	case CertSourceAutomate:
		if !replaceAutomateSubject(tls, index, req.Subject) {
			return fmt.Errorf("证书不存在")
		}
	}

	return s.client.ConfigLoad(ctx, cfg)
}

// CertDelete 删除证书
func (s *Service) CertDelete(ctx context.Context, key string) error {
	source, index, err := parseCertKey(key)
	if err != nil {
		return err
	}
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return err
	}
	tls := ensureTLS(cfg)

	switch source {
	case CertSourceFile:
		if tls.Certificates == nil || index < 0 || index >= len(tls.Certificates.LoadFiles) {
			return fmt.Errorf("证书不存在")
		}
		tls.Certificates.LoadFiles = append(tls.Certificates.LoadFiles[:index], tls.Certificates.LoadFiles[index+1:]...)
	case CertSourcePEM:
		if tls.Certificates == nil || index < 0 || index >= len(tls.Certificates.LoadPEM) {
			return fmt.Errorf("证书不存在")
		}
		tls.Certificates.LoadPEM = append(tls.Certificates.LoadPEM[:index], tls.Certificates.LoadPEM[index+1:]...)
	case CertSourceAutomate:
		if !removeAutomateSubject(tls, index) {
			return fmt.Errorf("证书不存在")
		}
	}

	return s.client.ConfigLoad(ctx, cfg)
}

// ─── 辅助：证书 ───

func buildCertKey(source string, index int) string {
	return fmt.Sprintf("%s-%d", source, index)
}

func parseCertKey(key string) (string, int, error) {
	parts := strings.SplitN(key, "-", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("无效的证书 key: %s", key)
	}
	source := parts[0]
	if source != CertSourceFile && source != CertSourcePEM && source != CertSourceAutomate {
		return "", 0, fmt.Errorf("不支持的证书来源: %s", source)
	}
	idx, err := strconv.Atoi(parts[1])
	if err != nil || idx < 0 {
		return "", 0, fmt.Errorf("无效的证书下标: %s", parts[1])
	}
	return source, idx, nil
}

// validateCertForm 校验证书表单
//
// create=true 时强制要求私钥非空；
// create=false（更新）时私钥留空表示保留原值，允许通过。
func validateCertForm(req CertForm, create bool) error {
	switch req.Source {
	case CertSourceFile:
		if strings.TrimSpace(req.Certificate) == "" {
			return fmt.Errorf("证书路径不能为空")
		}
		if create && strings.TrimSpace(req.KeyContent) == "" {
			return fmt.Errorf("私钥路径不能为空")
		}
	case CertSourcePEM:
		if strings.TrimSpace(req.Certificate) == "" {
			return fmt.Errorf("证书 PEM 内容不能为空")
		}
		if create && strings.TrimSpace(req.KeyContent) == "" {
			return fmt.Errorf("私钥 PEM 内容不能为空")
		}
	case CertSourceAutomate:
		if strings.TrimSpace(req.Subject) == "" {
			return fmt.Errorf("自动签发主机名不能为空")
		}
	default:
		return fmt.Errorf("不支持的证书来源: %s", req.Source)
	}
	return nil
}

func ensureCerts(tls *pkgcaddy.TLSApp) *pkgcaddy.TLSCerts {
	if tls.Certificates == nil {
		tls.Certificates = &pkgcaddy.TLSCerts{}
	}
	return tls.Certificates
}

// appendAutomateSubject 把 subject 追加到自动签发列表
//
// 如果 automation.policies 为空，创建一个新策略；否则追加到第一个策略
func appendAutomateSubject(tls *pkgcaddy.TLSApp, subject string) {
	if tls.Automation == nil {
		tls.Automation = &pkgcaddy.TLSAutomation{}
	}
	if len(tls.Automation.Policies) == 0 {
		tls.Automation.Policies = []pkgcaddy.TLSPolicy{{Subjects: []string{subject}}}
		return
	}
	tls.Automation.Policies[0].Subjects = append(tls.Automation.Policies[0].Subjects, subject)
}

// replaceAutomateSubject 按全局 index 替换 subject，返回是否成功
func replaceAutomateSubject(tls *pkgcaddy.TLSApp, index int, subject string) bool {
	if tls.Automation == nil {
		return false
	}
	cur := 0
	for pi := range tls.Automation.Policies {
		policy := &tls.Automation.Policies[pi]
		for si := range policy.Subjects {
			if cur == index {
				policy.Subjects[si] = subject
				return true
			}
			cur++
		}
	}
	return false
}

// removeAutomateSubject 按全局 index 删除 subject，返回是否成功
func removeAutomateSubject(tls *pkgcaddy.TLSApp, index int) bool {
	if tls.Automation == nil {
		return false
	}
	cur := 0
	for pi := range tls.Automation.Policies {
		policy := &tls.Automation.Policies[pi]
		for si := range policy.Subjects {
			if cur == index {
				policy.Subjects = append(policy.Subjects[:si], policy.Subjects[si+1:]...)
				return true
			}
			cur++
		}
	}
	return false
}
