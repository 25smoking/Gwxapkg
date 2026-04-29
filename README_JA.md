# Gwxapkg

<div align="center">

![Version](https://img.shields.io/badge/version-2.7.2-blue.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-00ADD8.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)
![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-lightgrey.svg)
![Build](https://img.shields.io/badge/build-passing-brightgreen.svg)

**[中文](README.md) | [English](README_EN.md) | [日本語](README_JA.md)**

Go で実装された WeChat ミニプログラム `.wxapkg` 解凍ツールです。自動スキャン、復号、逆コンパイル、セキュリティ分析に対応しています。

</div>

---

## ⚠️ 重要な法的および利用上の注意

**本ツールは、合法かつ正当で、かつ十分な権限を取得している場合に限り、セキュリティ調査、リバースエンジニアリング、互換性検証、学習交流、および自己所有または受託資産の技術監査のために使用することができます。**

**本ツールを使用する前に、すべての利用者は、対象となるミニプログラム、関連アカウント、端末、データ、ネットワーク環境、業務システム、その他の関連資産について、明確かつ継続的で、証明可能な法的権限を有していることを自ら確認しなければなりません。権限の範囲または有効性を確認できない場合は、直ちに使用を中止してください。**

### 禁止される用途

**本ツールは、無権限の利用や、法令、プラットフォーム規約、契約上の義務、知的財産権、またはプライバシー保護に違反するおそれのあるいかなる場面でも使用してはなりません。これには以下が含まれますが、これらに限られません。**

- **許可なく第三者のミニプログラムのコード、アセット、またはデータを解凍、解析、抽出、複製、配布すること**
- **プラットフォーム保護、リスク制御措置、アクセス制限、その他の技術的制御を回避すること**
- **大量収集、データスクレイピング、プライバシー情報の抽出、API の乱用、または自動化された攻撃的テスト**
- **商業的盗用、悪意ある模倣、悪意ある配布、不正産業での利用、その他の不適切な目的**
- **他者の権利侵害、プラットフォームからの制裁、業務中断、情報漏えい、またはコンプライアンスリスクを引き起こし得る行為**

### リスクに関する注意

**本ツールの利用には、法的責任、行政処分、民事賠償、刑事リスク、知的財産紛争、プライバシー侵害、アカウント停止、サービス中断、情報漏えい、本番事故、第三者からの請求などのリスクが伴う可能性があります。**

**利用者は、本ツールの使用、誤用、または乱用に起因する一切のリスクおよび結果を自ら評価し、自ら負担するものとします。**

### 責任の制限

**適用法令で認められる最大限の範囲において、本プロジェクトならびにその作者、コントリビューター、メンテナーは、本ツールの使用不能、使用、誤用、または乱用に起因するいかなる直接的、間接的、付随的、特別、懲罰的、または結果的損害についても責任を負いません。これには、データ損失、情報漏えい、プライバシー侵害、知的財産紛争、プラットフォーム制裁、アカウント停止、システム障害、業務損失、経済的損失、行政責任、刑事責任が含まれますが、これらに限られません。**

**本ツールは「現状有姿」で提供され、商品性、特定目的適合性、安定性、正確性、完全性、継続的利用可能性、適法性を含む一切の明示または黙示の保証を伴いません。**

### 使用継続は同意を意味します

**本ツールを引き続きダウンロード、インストール、実行、配布、または使用することにより、利用者は本通知の全文を読み、理解し、同意したものとみなされ、かつ合法的に権限を有する範囲内でのみ本ツールを使用することを約束するものとします。**

---

## ✨ 主な機能

### 🔍 スマート解凍
- **自動スキャン** - macOS / Windows の WeChat ミニプログラムキャッシュを自動検出
- **自動復号** - PC 版 WeChat キャッシュ内の暗号化 `wxapkg` を自動復号
- **ワンコマンド解凍** - 指定 AppID に属するすべてのパッケージを自動検出して処理
- **サブパッケージ対応** - メインパッケージとサブパッケージの依存関係を正しく処理

### 🎨 コード復元
- **完全復元** - `wxml` / `wxss` / `js` / `json` / `wxs` に対応
- **コード整形** - JavaScript / CSS / HTML を自動整形
- **既定反混淆** - 代表的な文字列配列、`\xNN`、`\uNNNN`、16 進数リテラルに対して静的復元と制御付きデコードを実施
- **ページルートマップ** - ページ一覧、エントリページ、分包、TabBar、コンポーネント依存、静的 / 動的遷移、トリガ手掛かり、ページ API 帰属を生成
- **ディレクトリ復元** - 元のミニプログラム構造に近い形へ復元
- **リソース抽出** - 画像、音声、動画などを完全抽出

### 🛡️ セキュリティ分析
- **スマートスキャン** - 既定で 920 件の敏感情報検出ルールを有効化
- **正式分類** - ルールをクラウド、決済、協作、監視、セキュリティ、SaaS などの正式カテゴリへ整理
- **誤検知抑制** - ブラックリスト、プレースホルダ、サンプル値、マスク値、弱値の多層フィルタ
- **重複排除** - 重複データを除去しつつ位置情報を保持
- **API 抽出** - URL / API Endpoint を抽出し、Postman Collection を出力可能
- **Excel / HTML レポート** - ファイルパスと行番号を含む Excel / HTML レポートを生成
- **リスク分類** - 高 / 中 / 低リスクを自動分類
- **混淆ファイル報告** - 混淆ファイルと復元状態を別枠で一覧化

### ⚡ パフォーマンス最適化
- **動的並列処理** - CPU コア数に応じて worker 数を自動調整
- **バッファ付き I/O** - 256 KB バッファで読書き性能を改善
- **ルール事前コンパイル** - 起動時に正規表現をまとめてコンパイル
- **最適化ビルド** - ビルド最適化でサイズ削減と速度改善

---

## 📊 対応ファイル形式

| ファイル形式 | 対応 | 説明 |
|-------------|------|------|
| `.wxml` | ✅ | ページ構造の復元 |
| `.wxss` | ✅ | スタイルの復元 |
| `.js` | ✅ | JavaScript 復元、整形、既定反混淆 |
| `.json` | ✅ | 設定抽出 |
| `.wxs` | ✅ | WXS 復元 |
| 画像 / 音声 / 動画 | ✅ | リソース抽出 |

---

## 📥 インストール

### 方法 1: ビルド済みバイナリをダウンロード

[Releases](https://github.com/25smoking/Gwxapkg/releases) ページから対象プラットフォーム向け実行ファイルを取得してください。

### 方法 2: ソースからビルド

```bash
# リポジトリを取得
git clone https://github.com/25smoking/Gwxapkg.git
cd Gwxapkg

# 最適化ビルド
go build -ldflags="-s -w" -o gwxapkg .

# 直接実行
go run . -h
```

**要件:** Go 1.21 以上

---

## 🚀 クイックスタート

### 基本的な使い方

```bash
# AppID を指定して自動スキャン + 自動処理
./gwxapkg all -id=<AppID>

# 利用可能なミニプログラム一覧を表示
./gwxapkg scan

# WeChat キャッシュ候補パスの診断を表示
./gwxapkg scan --verbose

# 単一 wxapkg を解凍
./gwxapkg -id=<AppID> -in=<file_path>

# 既に解凍済みのディレクトリを再スキャンし、Postman Collection も出力
./gwxapkg scan-only -dir=<directory> -format=both -postman

# 再パック
./gwxapkg repack -in=<directory_path>
```

### コマンド引数

| 引数 | 説明 | 既定値 |
|------|------|--------|
| `-id` | ミニプログラム AppID | - |
| `-in` | 入力ファイル / ディレクトリ | - |
| `-out` | 出力ディレクトリ | 自動 |
| `-restore` | プロジェクト構造を復元 | true |
| `-pretty` | 出力を整形 | true |
| `-sensitive` | 敏感情報スキャンを有効化 | true |
| `-postman` | `api_collection.postman_collection.json` を出力 | false |
| `-noClean` | 中間一時ファイルを保持 | false |
| `-save` | 復号後ファイルを保存 | false |
| `-workspace` | 正確な再パック用の隠しワークスペースを保持 | false |
| `--verbose` | WeChat キャッシュ候補パス診断を表示（`scan` / `all` のみ） | false |

### 使用例

```bash
# 例1: 自動スキャンして処理
./gwxapkg all -id=wx3c19e32cb8f31289

# 例2: 解凍して Postman Collection も出力
./gwxapkg all -id=wx3c19e32cb8f31289 -postman

# 例3: 単一ファイルを解凍
./gwxapkg -id=wx123456 -in=test.wxapkg -out=./output

# 例4: 解凍済みディレクトリを再スキャン
./gwxapkg scan-only -dir=./output/wx123456 -format=both -postman

# 例5: 再パック
./gwxapkg repack -in=./source_dir -out=new.wxapkg
```

### 既定の出力ディレクトリ

`-out` を指定しない場合、出力先は次のルールに従います。

- 正式ビルドされた実行ファイル: 実行ファイルの場所の下に `output/<AppID>`
- `go run .` または開発環境: 現在の作業ディレクトリの下に `output/<AppID>`
- 対話式 `scan` モード: 同じく `output/<AppID>`

例:

```text
/Applications/Gwxapkg/output/wx1234567890abcdef
./output/wx1234567890abcdef
```

### 典型的な出力構造

```text
output/
└── wx1234567890abcdef/
    ├── app.js
    ├── page-frame.html
    ├── sensitive_report.xlsx
    ├── sensitive_report.html
    ├── api_collection.postman_collection.json
    ├── route_manifest.json
    ├── route_map.md
    ├── route_map.mmd
    └── .gwxapkg/                   # -workspace=true のときのみ
```

---

## 📁 WeChat ミニプログラムキャッシュ位置

### macOS

```text
~/Library/Containers/com.tencent.xinWeChat/Data/Library/Caches/
├── applet/
│   ├── release/
│   └── debug/
└── ...
```

### Windows

```text
%USERPROFILE%\Documents\WeChat Files\Applet\
├── wx<appid>/
│   ├── <version>/
│   │   ├── __APP__.wxapkg
│   │   └── __SUBCONTEXT__.wxapkg
│   └── ...
└── ...
```

---

## 🎯 敏感情報スキャン

### スキャンルール（既定 920 ルール）

既定で 920 件の内蔵ルールが有効です。新規追加ルールの多くは依然として `xxx secret` のような命名ですが、プログラム内部では名称だけで粗く分類せず、サービス領域と用途を優先して正式カテゴリへ整理します。

| 分類 | ルール数 | 例 |
|------|----------|----|
| **第三者 SaaS** | 596 | 各種 `xxx secret`、サービス専用 token、Webhook、クライアント資格情報 |
| **Secret / Key** | 67 | 汎用 `client_secret`、`app_secret`、`session_secret` |
| **開発と配信** | 40 | GitHub、GitLab、NPM、PyPI、Jenkins、Terraform、JFrog |
| **クラウド** | 35 | AWS、阿里云、腾讯云、华为云、Azure、Cloudflare |
| **決済と EC** | 35 | Stripe、PayPal、Square、Razorpay、WeChat Pay、Shopify |
| **通知と協作** | 31 | Slack、Discord、Telegram、DingTalk、Feishu、Teams、Twilio |
| **監視とアラート** | 24 | Datadog、New Relic、Sentry、Grafana、PagerDuty |
| **Token / 令牌** | 16 | JWT、Bearer、OAuth Token、Session Token |
| **DB と接続** | 15 | MySQL、PostgreSQL、MongoDB、Redis、ES、InfluxDB |
| **パスワード** | 13 | 汎用パスワード、root / 管理者パスワード、DB / プロトコルパスワード |
| **セキュリティ基盤** | 13 | Auth0、Shodan、Censys、VirusTotal、AbuseIPDB |
| **秘密鍵と証明書** | 11 | RSA、DSA、EC、OpenSSH、PKCS8 秘密鍵と証明書 |
| **エンコードと指紋** | 8 | Hash、UUID、長い Base64、SSH 公開鍵 |
| **連絡先情報** | 3 | 電話番号、メールアドレス、身分証番号 |
| **ネットワーク識別子** | 3 | IPv4、内網 IP、MAC アドレス |
| **API Key** | 3 | 汎用 API Key / Access Key |
| **WeChat エコシステム** | 3 | AppID、CorpID、Secret |
| **URL / API** | 2 | URL、API Endpoint |
| **Path** | 1 | ファイルパス、システムパス |
| **Domain** | 1 | TLD 検証付きドメイン |

### 誤検知抑制戦略

920 ルールへ拡張した後も使いやすさを維持するため、スキャナは既定で次の収束処理を行います。

- **ブラックリストフィルタ** - よくある静的リソース名、フレームワーク API 断片、明らかな非機密文字列を除外
- **コンテキストフィルタ** - ドメインの TLD 検証、JS API 文脈判定、パス長検証を継続
- **プレースホルダフィルタ** - `your_api_key`、`replace_me`、`changeme`、`placeholder` などの例示値を除外
- **マスク値フィルタ** - `xxxxxx`、`******`、`<token>` などの伏字を除外
- **弱値フィルタ** - 資格情報系に最小長、文字形状、一般語判定を加えて弱い誤検知を抑制

### スキャンと出力の挙動

- `-sensitive=true` で `sensitive_report.xlsx` と `sensitive_report.html` を生成
- `-postman=true` で `api_collection.postman_collection.json` を生成
- `-postman` は `-sensitive` と独立して有効化可能
- `scan-only` は同じスキャナと JS 反混淆処理を再利用
- HTTP メソッドを安全に推定できない場合、Postman では `UNKNOWN` を出力
- 相対 API パスはそのまま保持し、`baseUrl` は自動補完しない

## 🧭 ページとルートマップ

解凍後の出力ディレクトリには次も生成されます。

- `route_manifest.json` - 機械可読なページ / ルート構造
- `route_map.md` - 人間向けのページ / ルート説明
- `route_map.mmd` - Mermaid で描画可能なルートグラフ

現在の復元能力:

- エントリページ判定
- 主包 / 分包ページ一覧
- TabBar ページ
- ページタイトルとファイル対応
- `usingComponents` 依存
- `wx.navigateTo` / `redirectTo` / `reLaunch` / `switchTab`
- WXML `<navigator>` の静的遷移
- WXML `bindtap` / `catchtap` の trigger -> handler 反引き
- `data-url` / `data-route` / `data-path` / `data-page` の補完
- テンプレート文字列、文字列連結、`dataset.url` 由来の動的ルート手掛かり
- ボタン文言 / trigger event / handler 名の関連付け
- ページスクリプト内の直接 API 帰属
- 共有サービスモジュール経由の間接 API 帰属
- ページ handler -> ローカル helper -> 共有 helper -> 最終遷移 の跨ファイル呼び出しチェーン復元
- `utils/router.js`、`common/nav.js` など共有ルート helper の識別と一覧化
- `app.json` に未宣言だが `Page(...)` を含む孤立ページ候補

### レポート内容

生成される Excel / HTML レポートには次が含まれます。

- **概要** - スキャン統計、リスク分布、カテゴリ集計
- **カテゴリ別ページ** - 実際の命中に応じて動的生成
- **URL / API** - URL、API Endpoint、周辺コンテキスト
- **DB と接続** - JDBC、MongoDB、Redis、ES など
- **Password / Secret / Token** - パスワード、汎用秘密情報、アクセス令牌
- **WeChat エコシステム** - WeChat 関連設定
- **混淆ファイル** - ファイルパス、スコア、技術、復元状態

各項目には以下が含まれます。
- 内容
- 出現回数
- ファイルパス
- 行番号
- リスクレベル

混淆ファイルにはさらに以下が含まれます。
- 状態（`restored` / `partial` / `flagged`）
- スコア
- 技術
- タグ（`[OBFUSCATED] ...`）

### Postman Collection 例

```json
{
  "info": {
    "name": "wx1234567890abcdef - API Collection"
  },
  "item": [
    {
      "name": "POST /api/user/login",
      "request": {
        "method": "POST",
        "url": {
          "raw": "/api/user/login"
        }
      }
    }
  ]
}
```

### ルール設定

- 既定では内蔵ルールセットを直接利用
- `config/rule.yaml` は自動生成しない
- 手動で `config/rule.yaml` を配置した場合は、その内容が内蔵ルールより優先
- ソースコードを変えずに監査方針だけ調整したい場合に適しています

---

## 📈 パフォーマンス比較（v2.7.2 vs v1.0）

| 指標 | v1.0 | v2.7.2 | 改善点 |
|------|------|--------|--------|
| **スキャン速度** | 基準 | +50-70% | 正規表現の事前コンパイル |
| **誤検知抑制** | 単純な正規表現走査 | 多層フィルタ | ブラックリスト + 文脈 + プレースホルダ + 弱値フィルタ |
| **データ量** | 127,185 件 | 約 3,000 件 | 重複排除 + フィルタ |
| **出力形式** | JSON | Excel / HTML | 対話的レポート |
| **並列性能** | 固定 10 worker | CPU*2 動的 | 自動適応 |
| **I/O 性能** | 直接書き込み | 256 KB バッファ | システムコール削減 |

---

## 🔄 バージョン更新

### v2.7.2 (2026-04-29) - 反難読化の安定性修正

#### 修正
- `all` 実行中に HTML の `<script>` 内 JavaScript を反難読化する際、`nil pointer dereference` で全体がクラッシュする可能性がある問題を修正
- `goja` AST の走査と変数宣言検出を強化し、不完全な `VariableStatement`、`Binding`、typed nil ノードを安全に扱うよう改善
- 単一の JavaScript 断片で反難読化分析に失敗しても、元の内容を保持して解包処理を継続するよう改善

#### 検証
- JavaScript 分析 panic、HTML 内蔵 script panic、不完全 AST ノードを対象に formatter 回帰テストを追加
- `go test ./...` と Windows amd64 クロスコンパイルで検証済み

### v2.7.1 (2026-04-21) - ルート解析強化

#### 新機能
- **跨ページ呼び出しチェーン復元** - ページ `handler -> ローカル helper -> 共有 helper -> 最終遷移` を `route_manifest.json` に記録
- **共有ルート helper 検出** - `utils/router.js` や `common/nav.js` などの共通遷移封装を認識し、使用ページ・方法・目標手掛かりを集計

#### 改良
- **ルートレポート強化** - `route_map.md` / `route_map.mmd` が呼び出しチェーンと共有 helper 情報を表示
- **フォールバック辺の整理** - 実際の遷移方法が復元できた場合は重複した `UNKNOWN` 辺を自動除去

### v2.7.0 (2026-04-18) - 大規模機能強化

#### 新機能
- **HTML 対話型レポート**
- **`scan-only` 独立モード**
- **GitHub Actions 統合**
- **Windows AppName 抽出**
- **バッチ処理拡張**
- **Postman Collection 出力**
- **既定 JS 反混淆**
- **混淆ファイル報告**
- **キャッシュパス診断**
- **内蔵ルール既定有効化**
- **ルールカテゴリ再整理**
- **誤検知抑制強化**

### v2.6.0 (2026-03-17) - 安定版

#### 新機能
- 安定性向上
- 継続的なルール最適化
- Windows / macOS / Linux 互換性向上
- 依存更新

#### 修正
- 一部サブパッケージ処理の不具合修正
- Excel レポートの特殊文字問題修正
- 大規模ミニプログラムでのメモリ使用量最適化

### v2.5.0 (2025-12-05) - 大型更新

#### 新機能
- マルチシート Excel レポート
- 誤検知抑制
- 重複排除
- リスク分類
- ファイルパスと行番号を含む完全コンテキスト

#### 改善
- 動的並列処理
- バッファ付き I/O
- ルール事前コンパイル
- 最適化ビルド

#### 修正
- ドメインルールがファイル名を誤検知する問題
- JavaScript API をドメイン扱いする問題
- ディレクトリ結合性能の改善

### v1.0.0 (2024-XX-XX)
- 初回リリース
- 基本解凍
- コード整形
- JSON 出力の敏感情報スキャン

---

## 🛠️ 技術アーキテクチャ

```text
Gwxapkg/
├── cmd/
│   └── root.go           # CLI 入口、進捗、レポート生成
├── internal/
│   ├── cmd/              # コマンド処理、ファイル解析
│   ├── decrypt/          # AES + XOR 復号
│   ├── unpack/           # wxapkg バイナリ解析
│   ├── restore/          # プロジェクト構造復元
│   ├── formatter/        # 整形と JS 反混淆
│   │   ├── jsformatter.go
│   │   └── deobfuscator.go
│   ├── key/              # ルール管理と事前コンパイル
│   ├── scanner/          # スキャンエンジン
│   │   ├── types.go
│   │   ├── rule_meta.go
│   │   ├── filter.go
│   │   ├── collector.go
│   │   ├── scanner.go
│   │   └── api_extractor.go
│   ├── reporter/         # レポート生成
│   │   ├── excel.go
│   │   ├── html.go
│   │   └── postman.go
│   ├── config/           # 設定管理
│   └── ui/               # ターミナル UI
├── config/
│   └── rule.yaml         # 任意のカスタム上書きファイル
└── main.go
```

---

## 🤝 コントリビューション

コントリビューション歓迎です。

1. このリポジトリを Fork
2. 機能ブランチを作成: `git checkout -b feature/AmazingFeature`
3. 変更をコミット: `git commit -m 'Add some AmazingFeature'`
4. ブランチを push: `git push origin feature/AmazingFeature`
5. Pull Request を作成

---

## 📄 ライセンス

本プロジェクトは MIT License で提供されています。詳細は [LICENSE](LICENSE) を参照してください。

---

## ❓ FAQ

### 1. ダブルクリックするとすぐ閉じるのはなぜですか？
このツールは CLI ツールであり、実行ファイルをダブルクリックして使う前提ではありません。

- 誤った方法: `gwxapkg.exe` をダブルクリック
- 正しい方法: ターミナルを開いてツールのディレクトリへ移動し、コマンドで実行

### 2. ミニプログラムのパッケージが見つからないのはなぜですか？
PC 版 WeChat にログイン済みで、対象ミニプログラムを少なくとも一度開いていることを確認してください。それでも見つからない場合は `scan` を実行し、検出されたキャッシュパスが正しいか確認してください。

---

## 📩 連絡先

WeChat で連絡する場合は、要件を明記してください。端末の開き方や Go のインストール方法などの基本的な質問には回答しません。
