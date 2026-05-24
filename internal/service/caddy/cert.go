package caddy

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	pkgcaddy "isrvd/pkgs/caddy"
)

// ─── TLS 证书 CRUD ───

// CertForm TLS 证书统一编辑模型
//
// 通过 Source 区分四种来源：
//   - file：load_files，Certificate/KeyContent 为磁盘文件路径
//   - pem：load_pem，Certificate/KeyContent 为 PEM 文本
//   - automate：automation.policies[].subjects，Subject 为域名
//   - cached：运行时文件缓存（只读），Subject 为域名，含有效期字段
type CertForm struct {
	Key         string   `json:"key,omitempty"`         // 复合主键 <source>-<index>，仅响应使用（cached 无此字段）
	Source      string   `json:"source"`                // file / pem / automate / cached
	Subject     string   `json:"subject,omitempty"`     // automate/cached：目标域名；file/pem：解析自证书 CN
	Certificate string   `json:"certificate,omitempty"` // file：证书文件路径；pem：证书 PEM 文本
	KeyContent  string   `json:"keyContent,omitempty"`  // file：私钥文件路径；pem：私钥 PEM 文本（响应时不返回）
	Tags        []string `json:"tags,omitempty"`        // Caddy 内部标签（file/pem 可选）
	Format      string   `json:"format,omitempty"`      // 证书格式，仅 file 使用（默认 PEM）

	// 以下字段由证书内容解析填充，automate 类型无证书文件故留空
	Issuer    string     `json:"issuer,omitempty"`    // 签发机构 Common Name
	NotBefore *time.Time `json:"notBefore,omitempty"` // 证书生效时间
	NotAfter  *time.Time `json:"notAfter,omitempty"`  // 证书过期时间
	SANs      []string   `json:"sans,omitempty"`      // Subject Alternative Names（DNS）
}

// CertList 列出所有证书（合并四种来源：file / pem / automate / cached）
func (s *Service) CertList(ctx context.Context) ([]CertForm, error) {
	cfg, err := s.client.ConfigAll(ctx)
	if err != nil {
		return nil, err
	}
	return s.certListFromConfig(cfg), nil
}

func (s *Service) certListFromConfig(cfg *pkgcaddy.Config) []CertForm {
	out := make([]CertForm, 0)
	if cfg.Apps != nil && cfg.Apps.TLS != nil {
		tls := cfg.Apps.TLS
		if tls.Certificates != nil {
			for i, f := range tls.Certificates.LoadFiles {
				form := CertForm{
					Key:         buildCertKey(CertSourceFile, i),
					Source:      CertSourceFile,
					Certificate: f.Certificate,
					Tags:        f.Tags,
					Format:      f.Format,
				}
				fillCertInfo(&form, parseCertFile(f.Certificate))
				out = append(out, form)
			}
			for i, p := range tls.Certificates.LoadPEM {
				form := CertForm{
					Key:         buildCertKey(CertSourcePEM, i),
					Source:      CertSourcePEM,
					Certificate: p.Certificate,
					Tags:        p.Tags,
				}
				fillCertInfo(&form, parseCertPEM([]byte(p.Certificate)))
				out = append(out, form)
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
	}

	// 追加运行时证书缓存（忽略扫描错误，不影响主列表）
	cached, _ := s.scanCertCache(cfg)
	out = append(out, cached...)
	return out
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

// ─── 证书缓存（运行时已签发证书，内部方法）───

// scanCertCache 扫描 Caddy storage root 下的证书缓存，返回 cached 类型的 CertForm 列表。
// 接收已读取的 cfg 避免重复请求。
//
// Caddy ACME 证书存储路径：<storage_root>/certificates/<acme_server_host>/<domain>/<domain>.crt
// 文件格式：PEM，包含私钥 + 证书链（多个 block），与 certify.certToPEM 输出格式一致。
func (s *Service) scanCertCache(cfg *pkgcaddy.Config) ([]CertForm, error) {
	if cfg.Storage == nil {
		return nil, nil
	}
	storageRoot, _ := cfg.Storage["root"].(string)
	if storageRoot == "" {
		return nil, nil
	}

	certsDir := filepath.Join(storageRoot, "certificates")
	var result []CertForm

	err := filepath.Walk(certsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".crt") {
			return nil
		}
		cert := parseCertFile(path)
		if cert == nil {
			return nil
		}
		rel, _ := filepath.Rel(storageRoot, path)
		form := CertForm{
			Key:    "cached-" + rel,
			Source: CertSourceCached,
		}
		fillCertInfo(&form, cert)
		result = append(result, form)
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("扫描证书缓存失败: %w", err)
	}
	return result, nil
}

// parseCertFile 从 PEM 文件中提取第一个 CERTIFICATE block 并解析。
// Caddy 缓存文件包含私钥 + 证书链多个 block，需逐块扫描。
func parseCertFile(path string) *x509.Certificate {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return parseCertPEM(data)
}

// parseCertPEM 从 PEM 字节中提取第一个 CERTIFICATE block 并解析。
// 支持多 block 文件（私钥 + 证书链），跳过非证书 block。
func parseCertPEM(data []byte) *x509.Certificate {
	for len(data) > 0 {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil
		}
		return cert
	}
	return nil
}

// fillCertInfo 将 x509.Certificate 中的证书信息填充到 CertForm
func fillCertInfo(form *CertForm, cert *x509.Certificate) {
	if cert == nil {
		return
	}
	if form.Subject == "" {
		form.Subject = cert.Subject.CommonName
		if form.Subject == "" && len(cert.DNSNames) > 0 {
			form.Subject = cert.DNSNames[0]
		}
	}
	form.Issuer = cert.Issuer.CommonName
	form.NotBefore = &cert.NotBefore
	form.NotAfter = &cert.NotAfter
	form.SANs = cert.DNSNames
}
