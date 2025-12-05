package key

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Rule struct {
	Id      string `yaml:"id"`
	Enabled bool   `yaml:"enabled"`
	Pattern string `yaml:"pattern"`
}

type Rules struct {
	Rules []Rule `yaml:"rules"`
}

func init() {
	configDir := "config"
	configFile := filepath.Join(configDir, "rule.yaml")

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Printf("Error creating config directory: %v\n", err)
			return
		}
		CreateConfigFile()
	}
}

func ReadRuleFile() (*Rules, error) {
	configFile := filepath.Join("config", "rule.yaml")
	file, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading rule file: %v", err)
	}

	var rules Rules
	if err := yaml.Unmarshal(file, &rules); err != nil {
		return nil, fmt.Errorf("error unmarshalling rule file: %v", err)
	}

	return &rules, nil
}

func CreateConfigFile() {
	configFile := filepath.Join("config", "rule.yaml")
	defaultRules := Rules{
		Rules: []Rule{
			// ============ 基础信息类 (20条) ============
			{Id: "email", Enabled: true, Pattern: `\b[A-Za-z0-9._\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,61}\b`},
			{Id: "phone_cn", Enabled: true, Pattern: `\b1[3-9]\d{9}\b`},
			{Id: "id_card_cn", Enabled: true, Pattern: `\b([1-9]\d{5}(19|20)\d{2}((0[1-9])|(1[0-2]))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx])\b`},
			{Id: "ipv4", Enabled: true, Pattern: `\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`},
			{Id: "path", Enabled: true, Pattern: `(?i)["']?((?:[/\\][\w.-]+)+(?:\.(?:exe|dll|so|dylib|sh|bat|cmd|ps1|py|js|php|asp|aspx|jsp|war|ear|class|jar|elf|com|sys|drv|vxd|ocx|cpl)))[\"']?`},
			{Id: "url", Enabled: true, Pattern: `(?i)(?:http[s]?://)([\w\-]+\.)+[\w\-]+(?:/[\w\-\./?%&=]*)?`},
			{Id: "api_endpoint", Enabled: true, Pattern: `(?i)["']?(?:https?://)?(?:[\w\-]+\.)+[\w\-]+/(?:api|v\d+)/[\w\-/]+["']?`},
			{Id: "domain", Enabled: true, Pattern: `\b(?:[a-zA-Z0-9](?:[a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}\b`},
			{Id: "internal_ip", Enabled: true, Pattern: `\b(?:10\.\d{1,3}\.\d{1,3}\.\d{1,3}|172\.(?:1[6-9]|2\d|3[01])\.\d{1,3}\.\d{1,3}|192\.168\.\d{1,3}\.\d{1,3})\b`},
			{Id: "mac_address", Enabled: true, Pattern: `\b(?:[0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}\b`},
			{Id: "credit_card", Enabled: true, Pattern: `\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|6(?:011|5[0-9]{2})[0-9]{12})\b`},
			{Id: "uuid", Enabled: true, Pattern: `\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`},
			{Id: "base64_long", Enabled: true, Pattern: `\b(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?(?:[A-Za-z0-9+/]{100,})\b`},
			{Id: "hex_key", Enabled: true, Pattern: `\b[0-9a-fA-F]{32,128}\b`},
			{Id: "ssh_rsa_public", Enabled: true, Pattern: `ssh-rsa\s+[A-Za-z0-9+/]+={0,2}`},
			{Id: "ssh_ed25519_public", Enabled: true, Pattern: `ssh-ed25519\s+[A-Za-z0-9+/]+={0,2}`},
			{Id: "md5_hash", Enabled: true, Pattern: `\b[a-fA-F0-9]{32}\b`},
			{Id: "sha1_hash", Enabled: true, Pattern: `\b[a-fA-F0-9]{40}\b`},
			{Id: "sha256_hash", Enabled: true, Pattern: `\b[a-fA-F0-9]{64}\b`},
			{Id: "json_web_token", Enabled: true, Pattern: `\beyJ[A-Za-z0-9_/+-]{10,}={0,2}\.[A-Za-z0-9_/+-]{15,}={0,2}\.[A-Za-z0-9_/+-]{10,}={0,2}\b`},

			// ============ 数据库连接类 (15条) ============
			{Id: "jdbc_mysql", Enabled: true, Pattern: `jdbc:mysql://[^\s"']+`},
			{Id: "jdbc_postgresql", Enabled: true, Pattern: `jdbc:postgresql://[^\s"']+`},
			{Id: "jdbc_oracle", Enabled: true, Pattern: `jdbc:oracle:thin:@[^\s"']+`},
			{Id: "jdbc_sqlserver", Enabled: true, Pattern: `jdbc:sqlserver://[^\s"']+`},
			{Id: "jdbc_db2", Enabled: true, Pattern: `jdbc:db2://[^\s"']+`},
			{Id: "mongodb_connection", Enabled: true, Pattern: `mongodb(?:\+srv)?://[^\s"']+`},
			{Id: "redis_connection", Enabled: true, Pattern: `redis://[^\s"']+`},
			{Id: "postgres_connection", Enabled: true, Pattern: `postgres(?:ql)?://[^\s"']+`},
			{Id: "mysql_connection", Enabled: true, Pattern: `mysql://[^\s"']+`},
			{Id: "db_username", Enabled: true, Pattern: `(?i)(?:db|database)[-_]?(?:user|username)["']?\s*[:=]\s*["']([a-zA-Z0-9_]{3,32})["']`},
			{Id: "db_password", Enabled: true, Pattern: `(?i)(?:db|database)[-_]?(?:pass|password|pwd)["']?\s*[:=]\s*["']([a-zA-Z0-9!@#$%^&*()_+\-=]{4,64})["']`},
			{Id: "db_host", Enabled: true, Pattern: `(?i)(?:db|database)[-_]?host["']?\s*[:=]\s*["']([a-zA-Z0-9.\-]+)["']`},
			{Id: "elasticsearch_url", Enabled: true, Pattern: `https?://[^\s"']*:9200`},
			{Id: "cassandra_connection", Enabled: true, Pattern: `cassandra://[^\s"']+`},
			{Id: "influxdb_connection", Enabled: true, Pattern: `https?://[^\s"']*:8086`},

			// ============ 密钥和私钥类 (18条) ============
			{Id: "private_key_rsa", Enabled: true, Pattern: `-----\s*?BEGIN\s*?RSA\s*?PRIVATE\s*?KEY\s*?-----[\s\S]*?-----\s*?END\s*?RSA\s*?PRIVATE\s*?KEY\s*?-----`},
			{Id: "private_key_dsa", Enabled: true, Pattern: `-----\s*?BEGIN\s*?DSA\s*?PRIVATE\s*?KEY\s*?-----[\s\S]*?-----\s*?END\s*?DSA\s*?PRIVATE\s*?KEY\s*?-----`},
			{Id: "private_key_ec", Enabled: true, Pattern: `-----\s*?BEGIN\s*?EC\s*?PRIVATE\s*?KEY\s*?-----[\s\S]*?-----\s*?END\s*?EC\s*?PRIVATE\s*?KEY\s*?-----`},
			{Id: "private_key_openssh", Enabled: true, Pattern: `-----\s*?BEGIN\s*?OPENSSH\s*?PRIVATE\s*?KEY\s*?-----[\s\S]*?-----\s*?END\s*?OPENSSH\s*?PRIVATE\s*?KEY\s*?-----`},
			{Id: "private_key_pkcs8", Enabled: true, Pattern: `-----\s*?BEGIN\s*?PRIVATE\s*?KEY\s*?-----[\s\S]*?-----\s*?END\s*?PRIVATE\s*?KEY\s*?-----`},
			{Id: "pgp_private_key", Enabled: true, Pattern: `-----\s*?BEGIN\s*?PGP\s*?PRIVATE\s*?KEY\s*?BLOCK\s*?-----[\s\S]*?-----\s*?END\s*?PGP\s*?PRIVATE\s*?KEY\s*?BLOCK\s*?-----`},
			{Id: "certificate", Enabled: true, Pattern: `-----\s*?BEGIN\s*?CERTIFICATE\s*?-----[\s\S]*?-----\s*?END\s*?CERTIFICATE\s*?-----`},
			{Id: "api_key_generic", Enabled: true, Pattern: `(?i)api[-_]?key["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{20,64})["']`},
			{Id: "secret_key_generic", Enabled: true, Pattern: `(?i)secret[-_]?key["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{20,64})["']`},
			{Id: "access_key_generic", Enabled: true, Pattern: `(?i)access[-_]?key["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{20,64})["']`},
			{Id: "encryption_key", Enabled: true, Pattern: `(?i)(?:encryption|encrypt)[-_]?key["']?\s*[:=]\s*["']([a-zA-Z0-9_\-+/=]{32,})["']`},
			{Id: "master_key", Enabled: true, Pattern: `(?i)master[-_]?key["']?\s*[:=]\s*["']([a-zA-Z0-9_\-+/=]{32,})["']`},
			{Id: "signing_key", Enabled: true, Pattern: `(?i)signing[-_]?key["']?\s*[:=]\s*["']([a-zA-Z0-9_\-+/=]{32,})["']`},
			{Id: "session_secret", Enabled: true, Pattern: `(?i)session[-_]?secret["']?\s*[:=]\s*["']([a-zA-Z0-9_\-+/=]{32,})["']`},
			{Id: "client_secret", Enabled: true, Pattern: `(?i)client[-_]?secret["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{32,})["']`},
			{Id: "app_secret", Enabled: true, Pattern: `(?i)app[-_]?secret["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{32,})["']`},
			{Id: "token_secret", Enabled: true, Pattern: `(?i)token[-_]?secret["']?\s*[:=]\s*["']([a-zA-Z0-9_\-+/=]{32,})["']`},
			{Id: "webhook_secret", Enabled: true, Pattern: `(?i)webhook[-_]?secret["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{20,64})["']`},

			// ============ 云服务商凭证 (25条) ============
			{Id: "aws_access_key_id", Enabled: true, Pattern: `\b((?:AKIA|ABIA|ACCA|ASIA)[0-9A-Z]{16})\b`},
			{Id: "aws_secret_access_key", Enabled: true, Pattern: `(?i)aws[-_]?secret[-_]?access[-_]?key["']?\s*[:=]\s*["']([A-Za-z0-9/+=]{40})["']`},
			{Id: "aws_session_token", Enabled: true, Pattern: `(?i)aws[-_]?session[-_]?token["']?\s*[:=]\s*["']([A-Za-z0-9/+=]{100,})["']`},
			{Id: "aws_account_id", Enabled: true, Pattern: `(?i)aws[-_]?account[-_]?id["']?\s*[:=]\s*["'](\d{12})["']`},
			{Id: "aws_arn", Enabled: true, Pattern: `arn:aws:[a-z0-9-]+:[a-z]{2}-[a-z]+-[0-9]+:\d+:.+`},
			{Id: "aws_s3_bucket", Enabled: true, Pattern: `s3://[a-z0-9._/-]+`},
			{Id: "aliyun_access_key", Enabled: true, Pattern: `\bLTAI[A-Za-z\d]{12,30}\b`},
			{Id: "tencent_secret_id", Enabled: true, Pattern: `\bAKID[A-Za-z\d]{13,40}\b`},
			{Id: "tencent_secret_key", Enabled: true, Pattern: `(?i)(?:tencent|qcloud)[-_]?secret[-_]?key["']?\s*[:=]\s*["']([A-Za-z0-9]{32,})["']`},
			{Id: "tencent_api_gateway", Enabled: true, Pattern: `\bAPID[a-zA-Z0-9]{32,42}\b`},
			{Id: "huawei_ak", Enabled: true, Pattern: `(?i)huawei[-_]?(?:ak|access[-_]?key)["']?\s*[:=]\s*["']([A-Z0-9]{20,})["']`},
			{Id: "google_api_key", Enabled: true, Pattern: `\bAIza[0-9A-Za-z_\-]{35}\b`},
			{Id: "google_oauth", Enabled: true, Pattern: `\bya29\.[0-9A-Za-z\-_]+\b`},
			{Id: "google_cloud_key", Enabled: true, Pattern: `(?i)(?:google|gcp)[-_]?(?:api|cloud)[-_]?key["']?\s*[:=]\s*["']([A-Za-z0-9_\-]{20,})["']`},
			{Id: "azure_client_secret", Enabled: true, Pattern: `(?i)azure[-_]?client[-_]?secret["']?\s*[:=]\s*["']([A-Za-z0-9_\-~.]{34,})["']`},
			{Id: "azure_tenant_id", Enabled: true, Pattern: `(?i)azure[-_]?tenant[-_]?id["']?\s*[:=]\s*["']([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})["']`},
			{Id: "volcengine_ak", Enabled: true, Pattern: `\b(?:AKLT|AKTP)[a-zA-Z0-9]{35,50}\b`},
			{Id: "kingsoft_ak", Enabled: true, Pattern: `\bAKLT[a-zA-Z0-9\-_]{16,28}\b`},
			{Id: "jdcloud_ak", Enabled: true, Pattern: `\bJDC_[0-9A-Z]{25,40}\b`},
			{Id: "baidu_ak", Enabled: true, Pattern: `(?i)baidu[-_]?(?:ak|api[-_]?key)["']?\s*[:=]\s*["']([A-Za-z0-9]{24,})["']`},
			{Id: "databricks_token", Enabled: true, Pattern: `\bdapi[a-f0-9]{32}\b`},
			{Id: "cloudflare_api_token", Enabled: true, Pattern: `(?i)cloudflare[-_]?(?:api[-_]?)?token["']?\s*[:=]\s*["']([A-Za-z0-9_\-]{40,})["']`},
			{Id: "digitalocean_token", Enabled: true, Pattern: `(?i)digitalocean[-_]?token["']?\s*[:=]\s*["']([A-Za-z0-9_\-]{64})["']`},
			{Id: "heroku_api_key", Enabled: true, Pattern: `(?i)heroku[-_]?api[-_]?key["']?\s*[:=]\s*["']([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})["']`},
			{Id: "ibm_cloud_api_key", Enabled: true, Pattern: `(?i)ibm[-_]?cloud[-_]?api[-_]?key["']?\s*[:=]\s*["']([A-Za-z0-9_\-]{44})["']`},

			// ============ 开发者工具和版本控制 (20条) ============
			{Id: "github_pat", Enabled: true, Pattern: `\bghp_[a-zA-Z0-9]{36,255}\b`},
			{Id: "github_oauth", Enabled: true, Pattern: `\bgho_[a-zA-Z0-9]{36,255}\b`},
			{Id: "github_app_token", Enabled: true, Pattern: `\b(ghu|ghs|ghr)_[a-zA-Z0-9]{36,255}\b`},
			{Id: "gitlab_pat", Enabled: true, Pattern: `\bglpat-[a-zA-Z0-9\-=_]{20,22}\b`},
			{Id: "gitlab_runner_token", Enabled: true, Pattern: `\bGR1348941[a-zA-Z0-9]{20}\b`},
			{Id: "bitbucket_token", Enabled: true, Pattern: `(?i)bitbucket[-_]?(?:token|key)["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{32,})["']`},
			{Id: "npm_token", Enabled: true, Pattern: `\bnpm_[a-zA-Z0-9]{36}\b`},
			{Id: "pypi_token", Enabled: true, Pattern: `\bpypi-AgEIcHlwaS5vcmc[A-Za-z0-9\-_]{50,1000}\b`},
			{Id: "docker_hub_token", Enabled: true, Pattern: `(?i)docker[-_]?(?:hub[-_]?)?(?:token|password)["']?\s*[:=]\s*["']([a-zA-Z0-9\-]{20,})["']`},
			{Id: "circleci_token", Enabled: true, Pattern: `(?i)circle[-_]?ci[-_]?token["']?\s*[:=]\s*["']([a-f0-9]{40})["']`},
			{Id: "travis_token", Enabled: true, Pattern: `(?i)travis[-_]?token["']?\s*[:=]\s*["']([a-zA-Z0-9]{22})["']`},
			{Id: "jenkins_token", Enabled: true, Pattern: `(?i)jenkins[-_]?token["']?\s*[:=]\s*["']([a-f0-9]{32,})["']`},
			{Id: "codecov_token", Enabled: true, Pattern: `(?i)codecov[-_]?token["']?\s*[:=]\s*["']([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})["']`},
			{Id: "sonar_token", Enabled: true, Pattern: `(?i)sonar[-_]?token["']?\s*[:=]\s*["']([a-f0-9]{40})["']`},
			{Id: "terraform_token", Enabled: true, Pattern: `\b[A-Za-z0-9]{14}\.atlasv1\.[A-Za-z0-9]{67}\b`},
			{Id: "ansible_vault", Enabled: true, Pattern: `\$ANSIBLE_VAULT;[0-9]\.[0-9];[A-Z0-9]+`},
			{Id: "jfrog_token", Enabled: true, Pattern: `(?i)jfrog[-_]?(?:token|api[-_]?key)["']?\s*[:=]\s*["']([A-Za-z0-9]{73,})["']`},
			{Id: "artifactory_token", Enabled: true, Pattern: `\b[A-Za-z0-9](?:[A-Za-z0-9\-]{0,61}[A-Za-z0-9])?\.jfrog\.io\b`},
			{Id: "maven_password", Enabled: true, Pattern: `(?i)<password>([^<]{8,})</password>`},
			{Id: "nuget_api_key", Enabled: true, Pattern: `(?i)nuget[-_]?api[-_]?key["']?\s*[:=]\s*["']([a-z0-9]{46})["']`},

			// ============ 支付和金融服务 (12条) ============
			{Id: "stripe_live_key", Enabled: true, Pattern: `\b(sk|rk)_live_[0-9a-zA-Z]{24,99}\b`},
			{Id: "stripe_test_key", Enabled: true, Pattern: `\b(sk|rk)_test_[0-9a-zA-Z]{24,99}\b`},
			{Id: "stripe_public_key", Enabled: true, Pattern: `\bpk_(?:live|test)_[0-9a-zA-Z]{24,99}\b`},
			{Id: "paypal_client_id", Enabled: true, Pattern: `(?i)paypal[-_]?client[-_]?id["']?\s*[:=]\s*["']([A-Za-z0-9\-_]{80})["']`},
			{Id: "paypal_secret", Enabled: true, Pattern: `(?i)paypal[-_]?secret["']?\s*[:=]\s*["']([A-Za-z0-9\-_]{80})["']`},
			{Id: "square_token", Enabled: true, Pattern: `\bsq0atp-[0-9A-Za-z\-_]{22,}\b`},
			{Id: "square_secret", Enabled: true, Pattern: `\bsq0csp-[0-9A-Za-z\-_]{43}\b`},
			{Id: "braintree_token", Enabled: true, Pattern: `\baccess_token\$production\$[0-9a-z]{16}\$[0-9a-f]{32}\b`},
			{Id: "razorpay_key", Enabled: true, Pattern: `\brzp_\w{2,6}_\w{10,20}\b`},
			{Id: "alipay_key", Enabled: true, Pattern: `(?i)alipay[-_]?(?:key|app[-_]?id)["']?\s*[:=]\s*["'](\d{16})["']`},
			{Id: "wechatpay_key", Enabled: true, Pattern: `(?i)wechat[-_]?pay[-_]?key["']?\s*[:=]\s*["']([a-zA-Z0-9]{32})["']`},
			{Id: "shopify_token", Enabled: true, Pattern: `\bshp(at|ca|pa|ss)_[a-fA-F0-9]{32}\b`},

			// ============ 通讯和消息服务 (18条) ============
			{Id: "slack_token", Enabled: true, Pattern: `\bxox[baprs]-[0-9a-zA-Z]{10,48}\b`},
			{Id: "slack_webhook", Enabled: true, Pattern: `https://hooks\.slack\.com/services/T[a-zA-Z0-9_]{8,10}/B[a-zA-Z0-9_]{8,12}/[a-zA-Z0-9_]{23,24}`},
			{Id: "discord_token", Enabled: true, Pattern: `\b[A-Za-z0-9_-]{24}\.[A-Za-z0-9_-]{6}\.[A-Za-z0-9_-]{27}\b`},
			{Id: "discord_webhook", Enabled: true, Pattern: `https://discord(?:app)?\.com/api/webhooks/\d{17,19}/[A-Za-z0-9_-]{60,68}`},
			{Id: "telegram_bot_token", Enabled: true, Pattern: `\b\d{8,10}:[a-zA-Z0-9_-]{35}\b`},
			{Id: "wechat_appid", Enabled: true, Pattern: `["']?(wx[a-z0-9]{15,18})["']?`},
			{Id: "wechat_corpid", Enabled: true, Pattern: `["']?(ww[a-z0-9]{15,18})["']?`},
			{Id: "wechat_secret", Enabled: true, Pattern: `(?i)wechat[-_]?(?:app)?[-_]?secret["']?\s*[:=]\s*["']([a-z0-9]{32})["']`},
			{Id: "wechat_webhook", Enabled: true, Pattern: `https://qyapi\.weixin\.qq\.com/cgi-bin/webhook/send\?key=[a-zA-Z0-9\-]{25,50}`},
			{Id: "dingtalk_webhook", Enabled: true, Pattern: `https://oapi\.dingtalk\.com/robot/send\?access_token=[a-z0-9]{50,80}`},
			{Id: "dingtalk_appkey", Enabled: true, Pattern: `(?i)dingtalk[-_]?app[-_]?key["']?\s*[:=]\s*["']([a-z0-9]{20,})["']`},
			{Id: "feishu_webhook", Enabled: true, Pattern: `https://open\.feishu\.cn/open-apis/bot/v2/hook/[a-z0-9\-]{25,50}`},
			{Id: "feishu_app_secret", Enabled: true, Pattern: `(?i)feishu[-_]?app[-_]?secret["']?\s*[:=]\s*["']([a-z0-9]{32})["']`},
			{Id: "twilio_account_sid", Enabled: true, Pattern: `\bAC[a-f0-9]{32}\b`},
			{Id: "twilio_auth_token", Enabled: true, Pattern: `(?i)twilio[-_]?auth[-_]?token["']?\s*[:=]\s*["']([a-f0-9]{32})["']`},
			{Id: "sendgrid_api_key", Enabled: true, Pattern: `\bSG\.[A-Za-z0-9_\-]{22,}\.[A-Za-z0-9_\-]{43}\b`},
			{Id: "mailgun_api_key", Enabled: true, Pattern: `\bkey-[0-9a-zA-Z]{32}\b`},
			{Id: "microsoft_teams_webhook", Enabled: true, Pattern: `https://[a-zA-Z0-9\-]+\.webhook\.office\.com/webhookb2/[a-zA-Z0-9\-@/]+`},

			// ============ 社交媒体和OAuth (10条) ============
			{Id: "facebook_access_token", Enabled: true, Pattern: `\bEAACEdEose0cBA[0-9A-Za-z]+\b`},
			{Id: "facebook_app_secret", Enabled: true, Pattern: `(?i)facebook[-_]?app[-_]?secret["']?\s*[:=]\s*["']([a-f0-9]{32})["']`},
			{Id: "twitter_api_key", Enabled: true, Pattern: `(?i)twitter[-_]?api[-_]?key["']?\s*[:=]\s*["']([a-zA-Z0-9]{25})["']`},
			{Id: "twitter_api_secret", Enabled: true, Pattern: `(?i)twitter[-_]?api[-_]?secret["']?\s*[:=]\s*["']([a-zA-Z0-9]{50})["']`},
			{Id: "twitter_bearer_token", Enabled: true, Pattern: `(?i)twitter[-_]?bearer["']?\s*[:=]\s*["']([A-Za-z0-9%\-_]{100,})["']`},
			{Id: "linkedin_client_secret", Enabled: true, Pattern: `(?i)linkedin[-_]?client[-_]?secret["']?\s*[:=]\s*["']([a-zA-Z0-9]{16})["']`},
			{Id: "instagram_access_token", Enabled: true, Pattern: `(?i)instagram[-_]?access[-_]?token["']?\s*[:=]\s*["']([0-9]+\.[a-f0-9]{32}\.[a-f0-9]{32})["']`},
			{Id: "oauth_client_secret", Enabled: true, Pattern: `(?i)(?:oauth|client)[-_]?secret["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{32,})["']`},
			{Id: "oauth_access_token", Enabled: true, Pattern: `(?i)(?:oauth[-_]?)?access[-_]?token["']?\s*[:=]\s*["']([a-zA-Z0-9_\-\.]{40,})["']`},
			{Id: "oauth_refresh_token", Enabled: true, Pattern: `(?i)refresh[-_]?token["']?\s*[:=]\s*["']([a-zA-Z0-9_\-\.]{40,})["']`},

			// ============ 认证和Token (20条) ============
			{Id: "bearer_token", Enabled: true, Pattern: `\b[Bb]earer\s+[a-zA-Z0-9\-=._+/\\]{20,500}\b`},
			{Id: "basic_auth", Enabled: true, Pattern: `\b[Bb]asic\s+[A-Za-z0-9+/]{18,}={0,2}\b`},
			{Id: "authorization_token", Enabled: true, Pattern: `['"]?[Aa]uthorization['"]?\s*[:=]\s*['"]?(?:[Tt]oken\s+)?[a-zA-Z0-9\-_+/]{20,500}['"]?`},
			{Id: "jwt_token_full", Enabled: true, Pattern: `eyJ[A-Za-z0-9_/+\-]{10,}={0,2}\.eyJ[A-Za-z0-9_/+\-]{10,}={0,2}\.[A-Za-z0-9_/+\-]{10,}={0,2}`},
			{Id: "api_token", Enabled: true, Pattern: `(?i)api[-_]?token["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{32,})["']`},
			{Id: "auth_token", Enabled: true, Pattern: `(?i)auth[-_]?token["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{32,})["']`},
			{Id: "session_token", Enabled: true, Pattern: `(?i)session[-_]?token["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{32,})["']`},
			{Id: "csrf_token", Enabled: true, Pattern: `(?i)csrf[-_]?token["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{32,})["']`},
			{Id: "xsrf_token", Enabled: true, Pattern: `(?i)xsrf[-_]?token["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{32,})["']`},
			{Id: "personal_access_token", Enabled: true, Pattern: `(?i)personal[-_]?access[-_]?token["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{40,})["']`},
			{Id: "service_account_token", Enabled: true, Pattern: `(?i)service[-_]?account[-_]?token["']?\s*[:=]\s*["']([a-zA-Z0-9_\-\.]{40,})["']`},
			{Id: "machine_token", Enabled: true, Pattern: `(?i)machine[-_]?token["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{40,})["']`},
			{Id: "deploy_token", Enabled: true, Pattern: `(?i)deploy[-_]?token["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{20,})["']`},
			{Id: "build_token", Enabled: true, Pattern: `(?i)build[-_]?token["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{20,})["']`},
			{Id: "ci_token", Enabled: true, Pattern: `(?i)ci[-_]?token["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{20,})["']`},
			{Id: "runner_token", Enabled: true, Pattern: `(?i)runner[-_]?(?:registration[-_]?)?token["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{20,})["']`},
			{Id: "pipeline_token", Enabled: true, Pattern: `(?i)pipeline[-_]?token["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{20,})["']`},
			{Id: "registry_token", Enabled: true, Pattern: `(?i)registry[-_]?token["']?\s*[:=]\s*["']([a-zA-Z0-9_\-\.]{40,})["']`},
			{Id: "encryption_token", Enabled: true, Pattern: `(?i)encryption[-_]?token["']?\s*[:=]\s*["']([a-zA-Z0-9_\-+/=]{32,})["']`},
			{Id: "verification_token", Enabled: true, Pattern: `(?i)verification[-_]?token["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{20,})["']`},

			// ============ 密码和账号 (12条) ============
			{Id: "password_generic", Enabled: true, Pattern: `(?i)(?:admin[-_]?pass|password|[a-z]{3,15}[-_]?password|user[-_]?pass|user[-_]?pwd|admin[-_]?pwd)\\?['"]?\s*[:=]\s*\\?['"]([a-zA-Z0-9!@#$%&*]{5,50})\\?['"]`},
			{Id: "username_password", Enabled: true, Pattern: `(?i)(?:user|username)["']?\s*[:=]\s*["']([a-zA-Z0-9_]{3,32})["'].*?(?:pass|password)["']?\s*[:=]\s*["']([^'"]{5,})["']`},
			{Id: "admin_password", Enabled: true, Pattern: `(?i)admin[-_]?(?:pass|password|pwd)["']?\s*[:=]\s*["']([a-zA-Z0-9!@#$%^&*()_+=\-]{5,64})["']`},
			{Id: "root_password", Enabled: true, Pattern: `(?i)root[-_]?(?:pass|password|pwd)["']?\s*[:=]\s*["']([a-zA-Z0-9!@#$%^&*()_+=\-]{5,64})["']`},
			{Id: "default_password", Enabled: true, Pattern: `(?i)default[-_]?(?:pass|password|pwd)["']?\s*[:=]\s*["']([a-zA-Z0-9!@#$%^&*]{5,64})["']`},
			{Id: "test_password", Enabled: true, Pattern: `(?i)test[-_]?(?:pass|password|pwd)["']?\s*[:=]\s*["']([a-zA-Z0-9!@#$%^&*]{5,64})["']`},
			{Id: "ftp_password", Enabled: true, Pattern: `(?i)ftp[-_]?(?:pass|password|pwd)["']?\s*[:=]\s*["']([a-zA-Z0-9!@#$%^&*]{5,64})["']`},
			{Id: "smtp_password", Enabled: true, Pattern: `(?i)smtp[-_]?(?:pass|password|pwd)["']?\s*[:=]\s*["']([a-zA-Z0-9!@#$%^&*]{5,64})["']`},
			{Id: "ldap_password", Enabled: true, Pattern: `(?i)ldap[-_]?(?:pass|password|pwd)["']?\s*[:=]\s*["']([a-zA-Z0-9!@#$%^&*]{5,64})["']`},
			{Id: "vpn_password", Enabled: true, Pattern: `(?i)vpn[-_]?(?:pass|password|pwd)["']?\s*[:=]\s*["']([a-zA-Z0-9!@#$%^&*]{5,64})["']`},
			{Id: "wifi_password", Enabled: true, Pattern: `(?i)(?:wifi|wireless)[-_]?(?:pass|password|pwd)["']?\s*[:=]\s*["']([a-zA-Z0-9!@#$%^&*]{8,64})["']`},
			{Id: "encryption_password", Enabled: true, Pattern: `(?i)encryption[-_]?(?:pass|password|pwd)["']?\s*[:=]\s*["']([a-zA-Z0-9!@#$%^&*]{8,64})["']`},

			// ============ 其他高价值服务 (20条) ============
			{Id: "postman_api_key", Enabled: true, Pattern: `\bPMAK-[a-zA-Z0-9]{59}\b`},
			{Id: "datadog_api_key", Enabled: true, Pattern: `(?i)datadog[-_]?api[-_]?key["']?\s*[:=]\s*["']([a-f0-9]{32})["']`},
			{Id: "newrelic_api_key", Enabled: true, Pattern: `\bNRAK-[A-Z0-9]{27}\b`},
			{Id: "sentry_dsn", Enabled: true, Pattern: `https://[a-f0-9]{32}@[a-z0-9.]+/\d+`},
			{Id: "bugsnag_api_key", Enabled: true, Pattern: `(?i)bugsnag[-_]?api[-_]?key["']?\s*[:=]\s*["']([a-f0-9]{32})["']`},
			{Id: "amplitude_api_key", Enabled: true, Pattern: `(?i)amplitude[-_]?api[-_]?key["']?\s*[:=]\s*["']([a-f0-9]{32})["']`},
			{Id: "segment_write_key", Enabled: true, Pattern: `(?i)segment[-_]?write[-_]?key["']?\s*[:=]\s*["']([a-zA-Z0-9]{32})["']`},
			{Id: "mixpanel_token", Enabled: true, Pattern: `(?i)mixpanel[-_]?token["']?\s*[:=]\s*["']([a-f0-9]{32})["']`},
			{Id: "grafana_api_key", Enabled: true, Pattern: `\beyJrIjoi[a-zA-Z0-9\-_+/]{50,100}={0,2}\b`},
			{Id: "grafana_cloud_token", Enabled: true, Pattern: `\bglc_[A-Za-z0-9\-_+/]{32,200}={0,2}\b`},
			{Id: "grafana_service_token", Enabled: true, Pattern: `\bglsa_[A-Za-z0-9]{32}_[A-Fa-f0-9]{8}\b`},
			{Id: "pagerduty_api_key", Enabled: true, Pattern: `(?i)pagerduty[-_]?api[-_]?key["']?\s*[:=]\s*["']([a-zA-Z0-9_\-+]{20})["']`},
			{Id: "opsgenie_api_key", Enabled: true, Pattern: `(?i)opsgenie[-_]?api[-_]?key["']?\s*[:=]\s*["']([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})["']`},
			{Id: "algolia_api_key", Enabled: true, Pattern: `(?i)algolia[-_]?(?:admin[-_]?)?api[-_]?key["']?\s*[:=]\s*["']([a-f0-9]{32})["']`},
			{Id: "mapbox_access_token", Enabled: true, Pattern: `\bpk\.[a-zA-Z0-9\-_]{60,}\b`},
			{Id: "shodan_api_key", Enabled: true, Pattern: `(?i)shodan[-_]?api[-_]?key["']?\s*[:=]\s*["']([a-zA-Z0-9]{32})["']`},
			{Id: "censys_api_key", Enabled: true, Pattern: `(?i)censys[-_]?api[-_]?(?:id|key)["']?\s*[:=]\s*["']([a-zA-Z0-9\-]{36})["']`},
			{Id: "virustotal_api_key", Enabled: true, Pattern: `(?i)virustotal[-_]?api[-_]?key["']?\s*[:=]\s*["']([a-f0-9]{64})["']`},
			{Id: "abuseipdb_api_key", Enabled: true, Pattern: `(?i)abuseipdb[-_]?api[-_]?key["']?\s*[:=]\s*["']([a-f0-9]{80})["']`},
			{Id: "notion_token", Enabled: true, Pattern: `\bsecret_[A-Za-z0-9]{43}\b`},
		},
	}

	data, err := yaml.Marshal(&defaultRules)
	if err != nil {
		fmt.Printf("Error marshalling default rules: %v\n", err)
		return
	}

	if err := os.WriteFile(configFile, data, 0755); err != nil {
		fmt.Printf("Error writing default rule file: %v\n", err)
	}
}
